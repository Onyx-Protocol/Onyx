package main

import (
	"context"
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
	"strings"
	"time"

	"github.com/kr/secureheader"
	"github.com/resonancelabs/go-pub/instrument"
	"github.com/resonancelabs/go-pub/instrument/client"

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
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
	"chain/errors"
	"chain/generated/dashboard"
	chainlog "chain/log"
	"chain/log/rotation"
	"chain/log/splunk"
	"chain/metrics"
	"chain/metrics/librato"
	"chain/net/http/gzip"
	"chain/net/http/httpspan"
	"chain/net/http/reqid"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
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
	// for config var LIBRATO_URL, see func init below
	traceguideToken  = os.Getenv("TRACEGUIDE_ACCESS_TOKEN")
	maxDBConns       = env.Int("MAXDBCONNS", 10) // set to 100 in prod
	apiSecretToken   = env.String("API_SECRET", "")
	rpcSecretToken   = env.String("RPC_SECRET", "secret")
	remoteSignerURLs = env.StringSlice("REMOTE_SIGNER_URLS")
	remoteSignerKeys = env.StringSlice("REMOTE_SIGNER_KEYS")

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
	librato.URL = env.URL("LIBRATO_URL", "")
	librato.Prefix = "chain.api."
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

	config, err := loadConfig(ctx, db)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}

	// Initialize internode rpc clients.
	hostname, err := os.Hostname()
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	processID := fmt.Sprintf("chain-%s-%s-%d", *target, hostname, os.Getpid())

	log.SetPrefix("api-" + buildTag + ": ")
	log.SetFlags(log.Lshortfile)
	chainlog.SetPrefix(append([]interface{}{"app", "api", "target", *target, "buildtag", buildTag, "processID", processID}, race...)...)
	chainlog.SetOutput(logWriter())

	requireSecretInProd(*apiSecretToken)

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

	var h http.Handler
	if config != nil {
		h = launchConfiguredCore(ctx, db, *config, processID)
	} else {
		chainlog.Messagef(ctx, "Launching as unconfigured Core.")
		h = core.Handler(*apiSecretToken, *rpcSecretToken, nil, nil, nil, nil, nil)
	}

	h = dashboardHandler(h)
	h = metrics.Handler{Handler: h}
	h = gzip.Handler{Handler: h}
	h = httpspan.Handler{Handler: h}
	h = dbContextHandler(h, db)
	h = reqid.Handler(h)
	http.Handle("/", h)
	http.HandleFunc("/health", func(http.ResponseWriter, *http.Request) {})
	secureheader.DefaultConfig.PermitClearLoopback = true
	secureheader.DefaultConfig.HTTPSRedirect = httpsRedirect

	server := &http.Server{
		Addr:    *listenAddr,
		Handler: secureheader.DefaultConfig,
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
	} else {
		err = server.ListenAndServe()
	}
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(err, "ListenAndServe"))
	}
}

func launchConfiguredCore(ctx context.Context, db *sql.DB, config core.Config, processID string) http.Handler {
	var remoteGenerator *rpc.Client
	if !config.IsGenerator {
		remoteGenerator = &rpc.Client{
			BaseURL:  config.GeneratorURL,
			Username: processID,
			BuildTag: buildTag,
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
	var signBlockHandler func(context.Context, *bc.Block) ([]byte, error)
	if config.IsSigner {
		var blockXPub *hd25519.XPub
		blockXPub, err = hd25519.XPubFromString(config.BlockXPub)
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
		for _, signer := range remoteSignerInfo(ctx, processID, buildTag, *rpcSecretToken) {
			generatorSigners = append(generatorSigners, signer)
		}
	}

	// Note, it's important for any services that will install blockchain
	// callbacks to be initialized before leader.Run() and the http server,
	// otherwise there's a data race within protocol.Chain.
	go leader.Run(db, *listenAddr, func(ctx context.Context) {
		ctx = pg.NewContext(ctx, db)

		// Must setup the indexer before generating or fetching blocks.
		err := indexer.BeginIndexing(ctx)
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}

		go utxodb.ExpireReservations(ctx, expireReservationsPeriod)
		if config.IsGenerator {
			go generator.Generate(ctx, c, generatorSigners, blockPeriod)
		} else {
			go fetch.Fetch(ctx, c, remoteGenerator)
		}
	})

	return core.Handler(*apiSecretToken, *rpcSecretToken, c, signBlockHandler, hsm, indexer, &config)
}

func dbContextHandler(handler http.Handler, db pg.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		ctx = pg.NewContext(ctx, db)
		handler.ServeHTTP(w, req.WithContext(ctx))
	})
}

// loadConfig loads the stored configuration, if any, from the database.
func loadConfig(ctx context.Context, db pg.DB) (*core.Config, error) {
	const q = `
		SELECT is_signer, is_generator, genesis_hash, generator_url, block_xpub, configured_at
		FROM config
	`

	c := new(core.Config)
	err := db.QueryRow(ctx, q).Scan(&c.IsSigner, &c.IsGenerator, &c.InitialBlockHash, &c.GeneratorURL, &c.BlockXPub, &c.ConfiguredAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "fetching Core config")
	}
	return c, nil
}

func dashboardHandler(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/dashboard/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		file := strings.TrimPrefix(req.URL.Path, "/dashboard/")
		output, ok := dashboard.Files[file]
		if !ok {
			output = dashboard.Files["index.html"]
		}
		http.ServeContent(w, req, file, time.Time{}, strings.NewReader(output))
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

func remoteSignerInfo(ctx context.Context, processID, buildTag, rpcSecretToken string) (a []*remoteSigner) {
	// REMOTE_SIGNER_URLS and REMOTE_SIGNER_KEYS should be parallel,
	// comma-separated lists. Each element of REMOTE_SIGNER_KEYS is the
	// public key for the corresponding URL in REMOTE_SIGNER_URLS.
	if len(*remoteSignerURLs) != len(*remoteSignerKeys) {
		chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(errors.New("REMOTE_SIGNER_URLS and REMOTE_SIGNER_KEYS must be same length")))
	}
	for i := range *remoteSignerURLs {
		u, err := url.Parse((*remoteSignerURLs)[i])
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}
		kbytes, err := hex.DecodeString((*remoteSignerKeys)[i])
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}
		k, err := hd25519.PubFromBytes(kbytes)
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(err), "at", "decoding signer public key")
		}
		client := &rpc.Client{
			BaseURL:  u.String(),
			Username: processID,
			BuildTag: buildTag,
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
