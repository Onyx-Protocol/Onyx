package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	_ "github.com/lib/pq"

	"chain/core/fetch"
	"chain/core/rpc"
	"chain/env"
	"chain/errors"
	"chain/log"
	"chain/net"
	"chain/net/http/httperror"
	"chain/net/http/httpjson"
	"chain/protocol"
	"chain/protocol/bc/legacy"
)

var (
	dbURL      = env.String("DATABASE_URL", "postgres:///blockcache?sslmode=disable")
	listen     = env.String("LISTEN", ":2000")
	target     = env.String("TARGET", "http://localhost:1999")
	targetAuth = env.String("TARGET_AUTH", "")
	tlsCrt     = env.String("TLSCRT", "")
	tlsKey     = env.String("TLSKEY", "")

	// aws relies on AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY being set
	region   = "us-east-1" // TODO(kr): figure out how to not hard code this
	awsSess  = session.Must(session.NewSession(aws.NewConfig().WithRegion(region)))
	awsS3    = s3.New(awsSess)
	s3Bucket = env.String("CACHEBUCKET", "blocks-cache")

	s3ACL      = aws.String("public-read")
	s3Encoding = aws.String("gzip")
	s3Type     = aws.String("application/json; charset=utf-8")
)

func main() {
	env.Parse()

	u, err := url.Parse(*target)
	if err != nil {
		log.Fatalkv(context.Background(), log.KeyError, err)
	}

	db, err := sql.Open("postgres", *dbURL)
	if err != nil {
		log.Fatalkv(context.Background(), log.KeyError, err)
	}

	var errorFormatter = httperror.Formatter{
		Default:     httperror.Info{500, "CH000", "Chain API Error"},
		IsTemporary: func(httperror.Info, error) bool { return false },
		Errors: map[error]httperror.Info{
			context.DeadlineExceeded: {408, "CH001", "Request timed out"},
			httpjson.ErrBadRequest:   {400, "CH003", "Invalid request body"},
		},
	}

	peer := &rpc.Client{
		BaseURL:     *target,
		AccessToken: *targetAuth,
	}

	const loadQ = `SELECT id, height FROM cache`
	var (
		id     string
		height uint64
	)
	err = db.QueryRow(loadQ).Scan(&id, &height)
	if err == sql.ErrNoRows {
		id, err = getBlockchainID(peer)
	}
	if err != nil {
		log.Fatalkv(context.Background(), log.KeyError, err)
	}

	peer.BlockchainID = id

	cache := &blockCache{
		db:     db,
		gz:     gzip.NewWriter(nil),
		id:     id,
		height: height,
	}
	cache.cond.L = &cache.mu

	go cacheBlocks(cache, peer)

	proxy := httputil.NewSingleHostReverseProxy(u)
	http.Handle("/", proxy)

	http.Handle("/rpc/get-block", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var height uint64
		err := json.NewDecoder(r.Body).Decode(&height)
		r.Body.Close()
		if err != nil {
			errorFormatter.Write(r.Context(), w, errors.WithDetail(httpjson.ErrBadRequest, err.Error()))
			return
		}

		ctx := r.Context()
		if t, err := time.ParseDuration(r.Header.Get(rpc.HeaderTimeout)); err == nil {
			var cancel func()
			ctx, cancel = context.WithTimeout(ctx, t)
			defer cancel()
		}

		err = cache.after(ctx, height)
		if err != nil {
			errorFormatter.Write(ctx, w, err)
			return
		}

		u := fmt.Sprintf("https://s3.amazonaws.com/%s/%s/%d", *s3Bucket, cache.getID(), height)
		http.Redirect(w, r, u, http.StatusFound)
	}))

	if *tlsCrt != "" {
		cert, err := tls.X509KeyPair([]byte(*tlsCrt), []byte(*tlsKey))
		if err != nil {
			log.Fatalkv(context.Background(), log.KeyError, errors.Wrap(err, "parsing tls X509 key pair"))
		}

		tlsConfig := net.DefaultTLSConfig()
		tlsConfig.Certificates = []tls.Certificate{cert}

		server := &http.Server{
			Addr:      *listen,
			Handler:   http.DefaultServeMux,
			TLSConfig: tlsConfig,
		}
		err = server.ListenAndServeTLS("", "")
		if err != nil {
			log.Error(context.Background(), err)
		}
	} else {
		err := http.ListenAndServe(*listen, http.DefaultServeMux)
		log.Error(context.Background(), err)
	}
}

type blockCache struct {
	cond   sync.Cond
	mu     sync.Mutex
	id     string
	height uint64

	db *sql.DB
	gz *gzip.Writer
}

func (c *blockCache) save(ctx context.Context, id string, height uint64, block *legacy.Block) error {
	buf := new(bytes.Buffer)
	c.gz.Reset(buf)
	err := json.NewEncoder(c.gz).Encode(block)
	if err != nil {
		return err
	}
	c.gz.Close()

	_, err = awsS3.PutObject(&s3.PutObjectInput{
		ACL:             s3ACL,
		Bucket:          s3Bucket,
		Key:             aws.String(fmt.Sprintf("%s/%d", id, height)),
		Body:            bytes.NewReader(buf.Bytes()),
		ContentEncoding: s3Encoding,
		ContentType:     s3Type,
	})
	if err != nil {
		return err
	}

	const q = `
		INSERT INTO cache (id, height) VALUES ($1, $2)
		ON CONFLICT (singleton) DO UPDATE
		SET id=EXCLUDED.id, height=EXCLUDED.height
	`
	_, err = c.db.Exec(q, id, height)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.id = id
	c.height = height
	c.mu.Unlock()
	c.cond.Broadcast()
	return nil
}

func (c *blockCache) after(ctx context.Context, height uint64) error {
	const slop = 3
	if height > c.getHeight()+slop {
		return protocol.ErrTheDistantFuture
	}

	errch := make(chan error)
	go func() {
		c.cond.L.Lock()
		defer c.cond.L.Unlock()
		for c.height < height { // c.height is safe to access since lock is held
			if height > c.height+slop { // due to reset
				errch <- protocol.ErrTheDistantFuture
				return
			}
			c.cond.Wait()
		}
		errch <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errch:
		return err
	}
}

func (c *blockCache) getHeight() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.height
}

func (c *blockCache) getID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.id
}

func cacheBlocks(cache *blockCache, peer *rpc.Client) {
	height := cache.getHeight() + 1
	ctx, cancel := context.WithCancel(context.Background())
	blocks, errs := fetch.DownloadBlocks(ctx, peer, height)
	for {
		select {
		case block := <-blocks:
			var nfailures uint
			for {
				err := cache.save(ctx, peer.BlockchainID, height, block)
				if err != nil {
					log.Error(ctx, err)
					nfailures++
					time.Sleep(backoffDur(nfailures))
				}
				break
			}
			height++
		case err := <-errs:
			if err == rpc.ErrWrongNetwork {
				cancel()
				peer.BlockchainID = "" // prevent ErrWrongNetwork
				peer.BlockchainID, err = getBlockchainID(peer)
				if err != nil {
					log.Fatalkv(ctx, log.KeyError, err)
				}
				height = 1

				ctx, cancel = context.WithCancel(context.Background())
				blocks, errs = fetch.DownloadBlocks(ctx, peer, height)
			} else {
				log.Fatalkv(ctx, log.KeyError, err)
			}
		}
	}
}

func getBlockchainID(peer *rpc.Client) (string, error) {
	var block *legacy.Block
	err := peer.Call(context.Background(), "/rpc/get-block", 1, &block)
	if err != nil {
		return "", err
	}
	h := block.Hash()
	return h.String(), nil
}

func backoffDur(n uint) time.Duration {
	if n > 33 {
		n = 33 // cap to about 10s
	}
	d := rand.Int63n(1 << n)
	return time.Duration(d)
}
