package main

import (
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

	"github.com/kr/secureheader"
	"github.com/resonancelabs/go-pub/instrument"
	"github.com/resonancelabs/go-pub/instrument/client"
	"golang.org/x/net/context"

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
	"chain/cos"
	"chain/cos/txscript"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
	"chain/errors"
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
	tlsCrt     = env.String("TLSCRT", "")
	tlsKey     = env.String("TLSKEY", "")
	listenAddr = env.String("LISTEN", ":8080")
	dbURL      = env.String("DB_URL", "postgres:///core?sslmode=disable")
	target     = env.String("TARGET", "sandbox")
	samplePer  = env.Duration("SAMPLEPER", 10*time.Second)
	splunkAddr = os.Getenv("SPLUNKADDR")
	logFile    = os.Getenv("LOGFILE")
	logSize    = env.Int("LOGSIZE", 5e6) // 5MB
	logCount   = env.Int("LOGCOUNT", 9)
	logQueries = env.Bool("LOG_QUERIES", false)
	blockKey   = env.String("BLOCK_KEY", "7a99f72169fad2d3a75aa36c550f60ee3a10f947ab5e4d38d5823667333d7e811af6c3e2396e20cab40770a8d8d5a906cb147539f390b57364b99b767d0b1418")
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
	sigsRequired       = env.Int("SIGS_REQUIRED", 1)

	// blockchain parameters
	maxProgramOps       = env.Int("MAX_PROGRAM_OPS", 1000)
	maxProgramStackSize = env.Int("MAX_PROGRAM_STACK", 1000)

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
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}

	txbuilder.Generator = remoteGeneratorURL

	privKey, err := hd25519.PrvFromBytes(keyBytes)
	if err != nil {
		panic(err)
	}
	pubKey := privKey.Public().(ed25519.PublicKey)

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

	// TODO(jackson): Propagate blockchain parameters to txscript via cos.FC.
	txscript.SetMaxStackSize(*maxProgramStackSize)
	txscript.SetMaxOpsPerScript(*maxProgramOps)

	sql.EnableQueryLogging(*logQueries)
	db, err := sql.Open("schemadb", *dbURL)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	db.SetMaxOpenConns(*maxDBConns)
	db.SetMaxIdleConns(100)
	ctx = pg.NewContext(ctx, db)
	heights, err := txdb.ListenBlocks(ctx, *dbURL)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	store, pool := txdb.New(db)
	fc, err := cos.NewFC(ctx, store, pool, []ed25519.PublicKey{pubKey}, heights)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}

	// Setup the transaction query indexer to index every transaction.
	indexer := query.NewIndexer(db, fc)
	indexer.RegisterAnnotator(account.AnnotateTxs)
	indexer.RegisterAnnotator(asset.AnnotateTxs)

	var localSigner *blocksigner.Signer
	if *isSigner {
		localSigner = blocksigner.New(privKey, db, fc)
	}

	rpcclient.Init(*remoteGeneratorURL)

	asset.Init(fc, indexer, *isManager)
	account.Init(fc)

	var generatorConfig *generator.Config
	if *isGenerator {
		remotes := remoteSignerInfo(ctx)
		nSigners := len(remotes)
		if *isSigner {
			nSigners++
		}
		if nSigners < *sigsRequired {
			chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(errors.New("too few signers configured")))
		}
		pubKeys := make([]ed25519.PublicKey, nSigners)
		for i, key := range remotes {
			pubKeys[i] = key.Key
		}
		if *isSigner {
			pubKeys[nSigners-1] = pubKey
		}

		generatorConfig = &generator.Config{
			RemoteSigners: remotes,
			LocalSigner:   localSigner,
			BlockPeriod:   blockPeriod,
			BlockKeys:     pubKeys,
			SigsRequired:  *sigsRequired,
			FC:            fc,
		}
	}

	// Note, it's important for any services that will install blockchain
	// callbacks to be initialized before leader.Run() and the http server,
	// otherwise there's a data race within cos.FC.
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
			go generator.Generate(ctx, *generatorConfig)
		} else {
			go fetch.Fetch(ctx, fc)
		}
	})

	hsm := mockhsm.New(db)

	h := core.Handler(*apiSecretToken, fc, generatorConfig, localSigner, store, pool, hsm, indexer)
	h = metrics.Handler{Handler: h}
	h = gzip.Handler{Handler: h}
	h = httpspan.Handler{Handler: h}

	http.Handle("/", chainhttp.ContextHandler{Context: ctx, Handler: h})
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
