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
	"chain/core/rpcclient"
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
)

var (
	// config vars
	tlsCrt       = env.String("TLSCRT", "")
	tlsKey       = env.String("TLSKEY", "")
	listenAddr   = env.String("LISTEN", ":8080")
	dbURL        = env.String("DATABASE_URL", "postgres:///core?sslmode=disable")
	target       = env.String("TARGET", "sandbox")
	samplePer    = env.Duration("SAMPLEPER", 10*time.Second)
	splunkAddr   = os.Getenv("SPLUNKADDR")
	logFile      = os.Getenv("LOGFILE")
	logSize      = env.Int("LOGSIZE", 5e6) // 5MB
	logCount     = env.Int("LOGCOUNT", 9)
	logQueries   = env.Bool("LOG_QUERIES", false)
	blockXPubStr = env.String("BLOCK_XPUB", "")
	// for config var LIBRATO_URL, see func init below
	traceguideToken    = os.Getenv("TRACEGUIDE_ACCESS_TOKEN")
	maxDBConns         = env.Int("MAXDBCONNS", 10) // set to 100 in prod
	apiSecretToken     = env.String("API_SECRET", "")
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

	race          []interface{} // initialized in race.go
	httpsRedirect = true        // initialized in insecure.go

	blockPeriod              = 1 * time.Second
	expireReservationsPeriod = time.Minute
)

// reserved mockhsm key alias
const autoBlockKeyAlias = "_CHAIN_CORE_AUTO_BLOCK_KEY"

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

	requireSecretInProd(*apiSecretToken)

	txbuilder.Generator = remoteGeneratorURL

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
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	db.SetMaxOpenConns(*maxDBConns)
	db.SetMaxIdleConns(100)
	ctx = pg.NewContext(ctx, db)

	hsm := mockhsm.New(db)

	var blockXPub *hd25519.XPub
	if *blockXPubStr == "" {
		coreXPub, created, err := hsm.GetOrCreateKey(ctx, autoBlockKeyAlias)
		if err != nil {
			panic(err)
		}
		blockXPub = coreXPub.XPub
		if created {
			log.Printf("Generated new block-signing key %s\n", blockXPub.String())
		} else {
			log.Printf("Using block-signing key %s\n", blockXPub.String())
		}
	} else {
		blockXPub, err = hd25519.XPubFromString(*blockXPubStr)
		if err != nil {
			panic(err)
		}
	}

	heights, err := txdb.ListenBlocks(ctx, *dbURL)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	store, pool := txdb.New(db)
	c, err := protocol.NewChain(ctx, store, pool, []ed25519.PublicKey{blockXPub.Key}, heights)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}

	// Setup the transaction query indexer to index every transaction.
	indexer := query.NewIndexer(db, c)
	indexer.RegisterAnnotator(account.AnnotateTxs)
	indexer.RegisterAnnotator(asset.AnnotateTxs)

	var localSigner *blocksigner.Signer
	if *isSigner {
		localSigner = blocksigner.New(blockXPub, hsm, db, c)
	}

	rpcclient.Init(*remoteGeneratorURL)

	asset.Init(c, indexer, *isManager)
	account.Init(c, indexer)

	var generatorConfig *generator.Config
	if *isGenerator {
		generatorConfig = &generator.Config{
			RemoteSigners: remoteSignerInfo(ctx),
			LocalSigner:   localSigner,
			Chain:         c,
		}
	}

	// Note, it's important for any services that will install blockchain
	// callbacks to be initialized before leader.Run() and the http server,
	// otherwise there's a data race within protocol.Chain.
	go leader.Run(db, func(ctx context.Context) {
		ctx = pg.NewContext(ctx, db)

		// Must setup the indexer before generating or fetching blocks.
		err := indexer.BeginIndexing(ctx)
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}

		if *isManager {
			go utxodb.ExpireReservations(ctx, expireReservationsPeriod)
		}
		if *isGenerator {
			go generator.Generate(ctx, *generatorConfig, blockPeriod)
		} else {
			go fetch.Fetch(ctx, c)
		}
	})

	h := core.Handler(*apiSecretToken, c, localSigner, store, pool, hsm, indexer)
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

func dbContextHandler(handler http.Handler, db pg.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		ctx = pg.NewContext(ctx, db)
		handler.ServeHTTP(w, req.WithContext(ctx))
	})
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

func remoteSignerInfo(ctx context.Context) (a []*generator.RemoteSigner) {
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
