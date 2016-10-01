package main

import (
	"context"
	"crypto/tls"
	"expvar"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kr/secureheader"

	"chain/core"
	"chain/core/account"
	"chain/core/account/utxodb"
	"chain/core/asset"
	"chain/core/blocksigner"
	"chain/core/fetch"
	"chain/core/generator"
	"chain/core/leader"
	"chain/core/mockhsm"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
	"chain/errors"
	"chain/generated/dashboard"
	chainlog "chain/log"
	"chain/log/rotation"
	"chain/log/splunk"
	"chain/net/http/gzip"
	"chain/net/http/reqid"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
)

const (
	httpReadTimeout  = 30 * time.Second
	httpWriteTimeout = 2 * time.Minute
)

var (
	// config vars
	tlsCrt     = env.String("TLSCRT", "")
	tlsKey     = env.String("TLSKEY", "")
	listenAddr = env.String("LISTEN", ":8080")
	dbURL      = env.String("DATABASE_URL", "postgres:///core?sslmode=disable")
	target     = env.String("TARGET", "sandbox")
	samplePer  = env.Duration("SAMPLEPER", 10*time.Second)
	splunkAddr = os.Getenv("SPLUNKADDR")
	logFile    = os.Getenv("LOGFILE")
	logSize    = env.Int("LOGSIZE", 5e6) // 5MB
	logCount   = env.Int("LOGCOUNT", 9)
	logQueries = env.Bool("LOG_QUERIES", false)
	maxDBConns = env.Int("MAXDBCONNS", 10) // set to 100 in prod

	// build vars; initialized by the linker
	buildTag    = "dev"
	buildCommit = "?"
	buildDate   = "?"

	race          []interface{} // initialized in race.go
	httpsRedirect = true        // initialized in insecure.go

	blockPeriod              = 1 * time.Second
	expireReservationsPeriod = time.Minute
)

func init() {
	expvar.NewString("buildtag").Set(buildTag)
	expvar.NewString("builddate").Set(buildDate)
	expvar.NewString("buildcommit").Set(buildCommit)
}

func main() {
	ctx := context.Background()
	env.Parse()

	sql.EnableQueryLogging(*logQueries)
	db, err := sql.Open("hapg", *dbURL)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	db.SetMaxOpenConns(*maxDBConns)
	db.SetMaxIdleConns(100)
	ctx = pg.NewContext(ctx, db)

	initSchemaInDev(db)
	resetInDevIfRequested(db)

	config, err := core.LoadConfig(ctx, db)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}

	// Initialize internode rpc clients.
	hostname, err := os.Hostname()
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	processID := fmt.Sprintf("chain-%s-%s-%d", *target, hostname, os.Getpid())

	log.SetPrefix("cored-" + buildTag + ": ")
	log.SetFlags(log.Lshortfile)
	chainlog.SetPrefix(append([]interface{}{"app", "cored", "target", *target, "buildtag", buildTag, "processID", processID}, race...)...)
	chainlog.SetOutput(logWriter())

	var h http.Handler
	if config != nil {
		h = launchConfiguredCore(ctx, db, config, processID)
	} else {
		chainlog.Messagef(ctx, "Launching as unconfigured Core.")
		h = core.Handler(nil, nil, nil, nil, nil)
	}

	h = dashboardHandler(h)
	h = gzip.Handler{Handler: h}
	h = dbContextHandler(h, db)
	h = reqid.Handler(h)
	h = timeoutContextHandler(h)
	http.Handle("/", h)
	http.HandleFunc("/health", func(http.ResponseWriter, *http.Request) {})
	secureheader.DefaultConfig.PermitClearLoopback = true
	secureheader.DefaultConfig.HTTPSRedirect = httpsRedirect

	server := &http.Server{
		Addr:         *listenAddr,
		Handler:      secureheader.DefaultConfig,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
	}
	if *tlsCrt != "" {
		cert, err := tls.X509KeyPair([]byte(*tlsCrt), []byte(*tlsKey))
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(err, "parsing tls X509 key pair"))
		}

		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		err = server.ListenAndServeTLS("", "") // uses TLS certs from above
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(err, "ListenAndServeTLS"))
		}
	} else {
		err = server.ListenAndServe()
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(err, "ListenAndServe"))
		}
	}
}

func launchConfiguredCore(ctx context.Context, db *sql.DB, config *core.Config, processID string) http.Handler {
	var remoteGenerator *rpc.Client
	if !config.IsGenerator {
		remoteGenerator = &rpc.Client{
			BaseURL:      config.GeneratorURL,
			Username:     processID,
			BuildTag:     buildTag,
			BlockchainID: config.BlockchainID.String(),
		}
	}
	txbuilder.Generator = remoteGenerator

	heights, err := txdb.ListenBlocks(ctx, *dbURL)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	store, pool := txdb.New(db)
	c, err := protocol.NewChain(ctx, store, pool, heights)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}

	// Setup the transaction query indexer to index every transaction.
	indexer := query.NewIndexer(db, c)
	indexer.RegisterAnnotator(account.AnnotateTxs)
	indexer.RegisterAnnotator(asset.AnnotateTxs)

	hsm := mockhsm.New(db)
	var generatorSigners []generator.BlockSigner
	var signBlockHandler core.BlockSignerFunc
	if config.IsSigner {
		var blockXPub chainkd.XPub
		err = blockXPub.UnmarshalText([]byte(config.BlockXPub))
		if err != nil {
			panic(err)
		}
		s := blocksigner.New(blockXPub, hsm, db, c)
		generatorSigners = append(generatorSigners, s) // "local" signer
		signBlockHandler = s.ValidateAndSignBlock
	}

	asset.Init(c, indexer)
	account.Init(c, indexer)

	if config.IsGenerator {
		for _, signer := range remoteSignerInfo(ctx, processID, buildTag, config.BlockchainID.String(), config) {
			generatorSigners = append(generatorSigners, signer)
		}
	}

	// Note, it's important for any services that will install blockchain
	// callbacks to be initialized before leader.Run() and the http server,
	// otherwise there's a data race within protocol.Chain.
	go leader.Run(db, *listenAddr, func(ctx context.Context) {
		ctx = pg.NewContext(ctx, db)

		go utxodb.ExpireReservations(ctx, expireReservationsPeriod)
		if config.IsGenerator {
			go generator.Generate(ctx, c, generatorSigners, blockPeriod)
		} else {
			go fetch.Fetch(ctx, c, remoteGenerator)
		}
	})

	h := core.Handler(c, signBlockHandler, hsm, indexer, config)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set(rpc.HeaderBlockchainID, config.BlockchainID.String())
		h.ServeHTTP(w, req)
	})
}

// timeoutContextHandler propagates the timeout, if any, provided as a header
// in the http request.
func timeoutContextHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		timeout, err := time.ParseDuration(req.Header.Get(rpc.HeaderTimeout))
		if err != nil {
			handler.ServeHTTP(w, req) // unmodified
			return
		}

		ctx := req.Context()
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		handler.ServeHTTP(w, req.WithContext(ctx))
	})
}

func dbContextHandler(handler http.Handler, db pg.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		ctx = pg.NewContext(ctx, db)
		handler.ServeHTTP(w, req.WithContext(ctx))
	})
}

func dashboardHandler(next http.Handler) http.Handler {
	lastMod := time.Now() // use start time as a conservative bound for last-modified
	mux := http.NewServeMux()
	mux.Handle("/dashboard/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		file := strings.TrimPrefix(req.URL.Path, "/dashboard/")
		output, ok := dashboard.Files[file]
		if !ok {
			output = dashboard.Files["index.html"]
		}
		http.ServeContent(w, req, file, lastMod, strings.NewReader(output))
	}))
	mux.Handle("/", next)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			http.Redirect(w, req, "/dashboard/", http.StatusFound)
			return
		}

		mux.ServeHTTP(w, req)
	})
}

// remoteSigner defines the address and public key of another Core
// that may sign blocks produced by this generator.
type remoteSigner struct {
	Client *rpc.Client
	Key    ed25519.PublicKey
}

func remoteSignerInfo(ctx context.Context, processID, buildTag, blockchainID string, config *core.Config) (a []*remoteSigner) {
	for _, signer := range config.Signers {
		u, err := url.Parse(signer.URL)
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}
		k, err := chainkd.NewEd25519PublicKey(signer.Pubkey)
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(err), "at", "decoding signer public key")
		}
		client := &rpc.Client{
			BaseURL:      u.String(),
			Username:     processID,
			BuildTag:     buildTag,
			BlockchainID: blockchainID,
		}
		a = append(a, &remoteSigner{Client: client, Key: k})
	}
	return a
}

func (s *remoteSigner) SignBlock(ctx context.Context, b *bc.Block) (signature []byte, err error) {
	// TODO(kr): We might end up serializing b multiple
	// times in multiple calls to different remoteSigners.
	// Maybe optimize that if it makes a difference.
	err = s.Client.Call(ctx, "/rpc/signer/sign-block", b, &signature)
	return
}

func (s *remoteSigner) String() string {
	return s.Client.BaseURL
}

func logWriter() io.Writer {
	dropmsg := []byte("\nlog data dropped\n")
	rotation := &errlog{w: rotation.Create(logFile, *logSize, *logCount)}
	splunk := &errlog{w: splunk.New(splunkAddr, dropmsg)}

	switch {
	case logFile != "" && splunkAddr != "":
		return io.MultiWriter(rotation, splunk)
	case logFile != "" && splunkAddr == "":
		return rotation
	case logFile == "" && splunkAddr != "":
		return splunk
	}
	return os.Stdout
}

type errlog struct {
	w io.Writer
	t time.Time // protected by chain/log mutex
}

func (w *errlog) Write(p []byte) (int, error) {
	// We don't want to ruin our performance
	// when there's a persistent error
	// writing to a log sink.
	// Print to stderr at most once per minute.
	_, err := w.w.Write(p)
	if err != nil && time.Since(w.t) > time.Minute {
		log.Println("chain/log:", err)
		w.t = time.Now()
	}
	return len(p), nil // report success for the MultiWriter
}
