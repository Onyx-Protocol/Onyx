package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"expvar"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/kr/secureheader"
	"github.com/resonancelabs/go-pub/instrument"
	"github.com/resonancelabs/go-pub/instrument/client"
	"golang.org/x/net/context"

	"chain/api"
	"chain/api/asset"
	"chain/api/generator"
	"chain/api/rpcclient"
	"chain/api/signer"
	"chain/api/smartcontracts/orderbook"
	"chain/api/txdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
	"chain/fedchain"
	chainlog "chain/log"
	"chain/log/rotation"
	"chain/log/splunk"
	"chain/metrics"
	"chain/metrics/librato"
	chainhttp "chain/net/http"
	"chain/net/http/gzip"
	"chain/net/http/httpspan"
	"chain/net/rpc"
)

var (
	// config vars
	tlsCrt       = env.String("TLSCRT", "")
	tlsKey       = env.String("TLSKEY", "")
	listenAddr   = env.String("LISTEN", ":8080")
	dbURL        = env.String("DB_URL", "postgres:///api?sslmode=disable")
	target       = env.String("TARGET", "sandbox")
	samplePer    = env.Duration("SAMPLEPER", 10*time.Second)
	nouserSecret = env.String("NOUSER_SECRET", "")
	splunkAddr   = os.Getenv("SPLUNKADDR")
	logFile      = os.Getenv("LOGFILE")
	logSize      = env.Int("LOGSIZE", 5e6) // 5MB
	logCount     = env.Int("LOGCOUNT", 9)
	logQueries   = env.Bool("LOG_QUERIES", false)
	blockKey     = env.String("BLOCK_KEY", "2c1f68880327212b6aa71d7c8e0a9375451143352d5c760dc38559f1159c84ce")
	// for config var LIBRATO_URL, see func init below
	traceguideToken    = os.Getenv("TRACEGUIDE_ACCESS_TOKEN")
	maxDBConns         = env.Int("MAXDBCONNS", 10) // set to 100 in prod
	rpcSecretToken     = env.String("RPC_SECRET", "secret")
	isSigner           = env.Bool("SIGNER", true) // node type must set FALSE explicitly
	isGenerator        = env.Bool("GENERATOR", true)
	isManager          = env.Bool("MANAGER", true)
	remoteGeneratorURL = env.String("REMOTE_GENERATOR_URL", "")
	remoteSignerURLs   = env.StringSlice("REMOTE_SIGNER_URLS")
	remoteSignerKeys   = env.StringSlice("REMOTE_SIGNER_KEYS")

	// build vars; initialized by the linker
	buildTag    = "dev"
	buildCommit = "?"
	buildDate   = "?"

	race []interface{} // initialized in race.go

	blockPeriod              = 1 * time.Second
	expireReservationsPeriod = time.Minute

	enableCrossProjectXferHack = env.Bool("ENABLE_CROSS_PROJECT_XFER_HACK", false)
)

func init() {
	librato.URL = env.URL("LIBRATO_URL", "")
	librato.Prefix = "chain.api."
	expvar.NewString("buildtag").Set(buildTag)
	expvar.NewString("builddate").Set(buildDate)
	expvar.NewString("buildcommit").Set(buildCommit)
}

func main() {
	ctx := context.Background()
	env.Parse()
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	// Initialize the internode rpc package.
	processID := fmt.Sprintf("chain-%s-%s-%d", *target, hostname, os.Getpid())
	rpc.LocalNode = rpc.NodeInfo{
		ProcessID: processID,
		Target:    *target,
		BuildTag:  buildTag,
	}
	rpc.SecretToken = *rpcSecretToken

	sql.Register("schemadb", pg.SchemaDriver(buildTag))
	log.SetPrefix("api-" + buildTag + ": ")
	log.SetFlags(log.Lshortfile)
	chainlog.SetPrefix(append([]interface{}{"app", "api", "target", *target, "buildtag", buildTag, "processID", processID}, race...)...)
	chainlog.SetOutput(logWriter())

	keyBytes, err := hex.DecodeString(*blockKey)
	if err != nil {
		chainlog.Fatal(ctx, "error", err)
	}

	asset.Generator = remoteGeneratorURL

	privKey, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), keyBytes)

	if librato.URL.Host != "" {
		librato.Source = *target
		go librato.SampleMetrics(*samplePer)
	} else {
		log.Println("no metrics; set LIBRATO_URL for prod")
	}

	if traceguideToken == "" {
		log.Println("no tracing; set TRACEGUIDE_ACCESS_TOKEN for prod")
	}
	instrument.SetDefaultRuntime(client.NewRuntime(&client.Options{
		AccessToken: traceguideToken,
		GroupName:   "api",
		Attributes: map[string]interface{}{
			"target":      *target,
			"buildtag":    buildTag,
			"builddate":   buildDate,
			"buildcommit": buildCommit,
		},
	}))

	sql.EnableQueryLogging(*logQueries)
	db, err := sql.Open("schemadb", *dbURL)
	if err != nil {
		chainlog.Fatal(ctx, "error", err)
	}
	db.SetMaxOpenConns(*maxDBConns)
	db.SetMaxIdleConns(100)
	ctx = pg.NewContext(ctx, db)

	fc, err := fedchain.New(ctx, txdb.NewStore(), []*btcec.PublicKey{pubKey})
	if err != nil {
		chainlog.Fatal(ctx, "error", err)
	}

	var localSigner *signer.Signer
	if *isSigner {
		localSigner = signer.New(privKey, fc)
	}

	rpcclient.Init(fc, *remoteGeneratorURL)

	asset.Init(fc, *isManager)

	if *isManager {
		orderbook.ConnectFedchain(fc)
	}

	if *isGenerator {
		remotes := remoteSignerInfo(ctx)
		err := generator.Init(ctx, fc, []*btcec.PublicKey{pubKey}, 1, blockPeriod, localSigner, remotes)
		if err != nil {
			chainlog.Fatal(ctx, "error", err)
		}
	}

	go determineLeader(ctx)

	h := api.Handler(*nouserSecret, localSigner)
	h = metrics.Handler{Handler: h}
	h = gzip.Handler{Handler: h}
	h = httpspan.Handler{Handler: h}

	http.Handle("/", chainhttp.ContextHandler{Context: ctx, Handler: h})
	http.HandleFunc("/health", func(http.ResponseWriter, *http.Request) {})

	secureheader.DefaultConfig.PermitClearLoopback = true
	api.EnableCrossProjectXferHack = *enableCrossProjectXferHack

	server := &http.Server{
		Addr:    *listenAddr,
		Handler: secureheader.DefaultConfig,
	}
	if *tlsCrt != "" {
		cert, err := tls.X509KeyPair([]byte(*tlsCrt), []byte(*tlsKey))
		if err != nil {
			chainlog.Fatal(ctx, "error", "parsing tls X509 key pair", err.Error())
		}

		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		err = server.ListenAndServeTLS("", "") // uses TLS certs from above
	} else {
		err = server.ListenAndServe()
	}
	if err != nil {
		chainlog.Fatal(ctx, "error", "ListenAndServe", err.Error())
	}
}

func remoteSignerInfo(ctx context.Context) (a []*generator.RemoteSigner) {
	// REMOTE_SIGNER_URLS and REMOTE_SIGNER_KEYS should be parallel,
	// comma-separated lists. Each element of REMOTE_SIGNER_KEYS is the
	// public key for the corresponding URL in REMOTE_SIGNER_URLS.
	if len(*remoteSignerURLs) != len(*remoteSignerKeys) {
		chainlog.Fatal(ctx, "error", "REMOTE_SIGNER_URLS and REMOTE_SIGNER_KEYS must be same length")
	}
	for i := range *remoteSignerURLs {
		u, err := url.Parse((*remoteSignerURLs)[i])
		if err != nil {
			chainlog.Fatal(ctx, "error", err)
		}
		b, err := hex.DecodeString((*remoteSignerKeys)[i])
		if err != nil {
			chainlog.Fatal(ctx, "error", err, "at", "decoding signer public key")
		}
		k, err := btcec.ParsePubKey(b, btcec.S256())
		if err != nil {
			chainlog.Fatal(ctx, "error", err, "at", "parsing signer public key")
		}
		a = append(a, &generator.RemoteSigner{URL: u, Key: k})
	}
	return a
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

// This runs as a goroutine, trying once every five seconds to become
// the leader for the core.  If it succeeds, then it launches the
// leader goroutines (for generating or fetching blocks, and for
// expiring reservations) and enters a leadership-keepalive loop.
//
// The Chain Core has up to a 10-second refractory period after
// shutdown, during which no process can become the new leader.
func determineLeader(ctx context.Context) {
	leaderKeyBytes := make([]byte, 32)
	_, err := rand.Read(leaderKeyBytes)
	if err != nil {
		chainlog.Fatal(ctx, "error", err)
	}
	leaderKey := hex.EncodeToString(leaderKeyBytes)
	log.Println("Chose leaderKey:", leaderKey)

	const (
		insertQ = `
			INSERT INTO leader (leader_key, expiry) VALUES ($1, CURRENT_TIMESTAMP + INTERVAL '10 seconds')
			ON CONFLICT (singleton) DO UPDATE SET leader_key = $1, expiry = CURRENT_TIMESTAMP + INTERVAL '10 seconds'
				WHERE leader.expiry < CURRENT_TIMESTAMP
		`
		updateQ = `
			UPDATE leader SET expiry = CURRENT_TIMESTAMP + INTERVAL '10 seconds'
				WHERE leader_key = $1
		`
	)

	var deposed chan struct{}
	leading := false

	for range time.Tick(5 * time.Second) {
		if leading {
			res, err := pg.Exec(ctx, updateQ, leaderKey)
			if err == nil {
				rowsAffected, err := res.RowsAffected()
				if err == nil && rowsAffected > 0 {
					// still leading
					continue
				}
			}

			// Either the UPDATE affected no rows, or it (or RowsAffected)
			// produced an error.

			if err != nil {
				chainlog.Error(ctx, err)
			}
			log.Println("No longer core leader")
			close(deposed)
			leading = false
		} else {
			// Try to put this process's leaderKey into the leader table.  It
			// succeeds if the table's empty or the existing row (there can be
			// only one) is expired.  It fails otherwise.
			//
			// On success, this process's leadership expires in 10 seconds
			// unless it's renewed using maintain_leadership() (which happens
			// below).  That extends it for another 10 seconds.
			res, err := pg.Exec(ctx, insertQ, leaderKey)
			if err != nil {
				chainlog.Error(ctx, err)
				continue
			}
			rowsAffected, err := res.RowsAffected()
			if err != nil {
				chainlog.Error(ctx, err)
				continue
			}

			if rowsAffected == 0 {
				continue
			}

			log.Println("I am the core leader")

			deposed = make(chan struct{})
			leading = true

			go blockLoop(ctx, deposed)
			if *isManager {
				go utxodb.ExpireReservations(ctx, expireReservationsPeriod, deposed)
			}
		}
	}
}

func blockLoop(ctx context.Context, deposed <-chan struct{}) {
	if generator.Enabled() {
		err := generator.UpsertGenesisBlock(ctx)
		if err != nil {
			panic(err)
		}
	}

	ticks := time.Tick(blockPeriod)
	for {
		select {
		case <-deposed:
			log.Println("Deposed, blockLoop exiting")
			return
		case <-ticks:
			func() {
				defer chainlog.RecoverAndLogError(ctx)
				var err error
				if *isGenerator {
					_, err = generator.MakeBlock(ctx)
				} else {
					err = rpcclient.GetBlocks(ctx)
				}
				if err != nil {
					chainlog.Error(ctx, err)
				}
			}()
		}
	}
}
