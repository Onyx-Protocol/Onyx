// Command cored provides the Chain Core daemon and API server.
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"expvar"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"chain/core"
	"chain/core/accesstoken"
	"chain/core/account"
	"chain/core/asset"
	"chain/core/blocksigner"
	"chain/core/config"
	"chain/core/fetch"
	"chain/core/generator"
	"chain/core/leader"
	"chain/core/migrate"
	"chain/core/mockhsm"
	"chain/core/pb"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/rpc"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/core/txfeed"
	"chain/crypto/ed25519"
	"chain/database/sql"
	"chain/env"
	"chain/errors"
	chainlog "chain/log"
	"chain/log/rotation"
	"chain/log/splunk"
	"chain/protocol"
	"chain/protocol/bc"
)

const (
	httpReadTimeout  = 2 * time.Minute
	httpWriteTimeout = time.Hour
	latestVersion    = "1.0.2"
)

var (
	// config vars
	tlsCrt        = env.String("TLSCRT", "")
	tlsKey        = env.String("TLSKEY", "")
	listenAddr    = env.String("LISTEN", ":1999")
	dbURL         = env.String("DATABASE_URL", "postgres:///core?sslmode=disable")
	splunkAddr    = os.Getenv("SPLUNKADDR")
	logFile       = os.Getenv("LOGFILE")
	logSize       = env.Int("LOGSIZE", 5e6) // 5MB
	logCount      = env.Int("LOGCOUNT", 9)
	logQueries    = env.Bool("LOG_QUERIES", false)
	maxDBConns    = env.Int("MAXDBCONNS", 10)           // set to 100 in prod
	rpsToken      = env.Int("RATELIMIT_TOKEN", 0)       // reqs/sec
	rpsRemoteAddr = env.Int("RATELIMIT_REMOTE_ADDR", 0) // reqs/sec
	indexTxs      = env.Bool("INDEX_TRANSACTIONS", true)

	// build vars; initialized by the linker
	buildTag    = "dev"
	buildCommit = "?"
	buildDate   = "?"

	race          []interface{} // initialized in race.go
	httpsRedirect = true        // initialized in insecure.go

	blockPeriod              = time.Second
	expireReservationsPeriod = time.Second
)

func init() {
	var version string
	if strings.HasPrefix(buildTag, "cmd.cored-") {
		// build tag with cmd.cored- prefix indicates official release
		version = latestVersion
	} else if buildTag != "?" {
		version = latestVersion + "-" + buildTag
	} else {
		// -dev suffix indicates intermediate, non-release build
		version = latestVersion + "-dev"
	}

	expvar.NewString("prod").Set(prod)
	expvar.NewString("version").Set(version)
	expvar.NewString("buildtag").Set(buildTag)
	expvar.NewString("builddate").Set(buildDate)
	expvar.NewString("buildcommit").Set(buildCommit)
	expvar.NewString("runtime.GOOS").Set(runtime.GOOS)
	expvar.NewString("runtime.GOARCH").Set(runtime.GOARCH)
	expvar.NewString("runtime.Version").Set(runtime.Version())

	config.Version = version
	config.BuildCommit = buildCommit
	config.BuildDate = buildDate
}

func main() {
	maybeMonitorIfOnWindows() // special-case windows

	ctx := context.Background()
	env.Parse()

	sql.EnableQueryLogging(*logQueries)
	db, err := sql.Open("hapg", *dbURL)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	db.SetMaxOpenConns(*maxDBConns)
	db.SetMaxIdleConns(*maxDBConns)

	err = migrate.Run(db)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	resetInDevIfRequested(db)

	conf, err := config.Load(ctx, db)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}

	// Initialize internode rpc clients.
	hostname, err := os.Hostname()
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	processID := fmt.Sprintf("chain-%s-%d", hostname, os.Getpid())
	if conf != nil {
		processID += "-" + conf.ID
	}
	expvar.NewString("processID").Set(processID)

	log.SetPrefix("cored-" + buildTag + ": ")
	log.SetFlags(log.Lshortfile)
	chainlog.SetPrefix(append([]interface{}{"app", "cored", "buildtag", buildTag, "processID", processID}, race...)...)
	chainlog.SetOutput(logWriter())

	var h *core.Handler
	if conf != nil {
		h = launchConfiguredCore(ctx, db, conf, processID)
	} else {
		chainlog.Messagef(ctx, "Launching as unconfigured Core.")
		h = &core.Handler{
			DB:           db,
			AltAuth:      authLoopbackInDev,
			AccessTokens: &accesstoken.CredentialStore{DB: db},
		}
	}

	// Give the remainder of this function a second to reach the
	// ListenAndServe call, then log a welcome message.
	go func() {
		time.Sleep(time.Second)
		chainlog.Messagef(ctx, "Chain Core online and listening at %s", *listenAddr)
	}()

	var cert *tls.Certificate
	if *tlsCrt != "" {
		tlsCert, err := tls.X509KeyPair([]byte(*tlsCrt), []byte(*tlsKey))
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(err, "parsing tls X509 key pair"))
		}
		cert = &tlsCert
	}

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(err, "listening on LISTEN"))
	}
	server := h.Server(cert)
	err = server.Serve(listener)
	chainlog.Fatal(ctx, chainlog.KeyError, err)
}

func launchConfiguredCore(ctx context.Context, db *sql.DB, conf *config.Config, processID string) *core.Handler {
	// Initialize the protocol.Chain.
	heights, err := txdb.ListenBlocks(ctx, *dbURL)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	store := txdb.NewStore(db)
	c, err := protocol.NewChain(ctx, conf.BlockchainID, store, heights)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}

	hsm := mockhsm.New(db)

	var generatorSigners []generator.BlockSigner
	var signBlockHandler func(context.Context, *bc.Block) ([]byte, error)
	if conf.IsSigner {
		s := blocksigner.New(conf.BlockPub, hsm, db, c)
		generatorSigners = append(generatorSigners, s) // "local" signer
		signBlockHandler = func(ctx context.Context, b *bc.Block) ([]byte, error) {
			sig, err := s.ValidateAndSignBlock(ctx, b)
			if errors.Root(err) == blocksigner.ErrInvalidKey {
				chainlog.Fatal(ctx, chainlog.KeyError, err)
			}
			return sig, err
		}
	}
	if conf.IsGenerator {
		for _, signer := range remoteSignerInfo(ctx, processID, buildTag, conf.BlockchainID.String(), conf) {
			generatorSigners = append(generatorSigners, signer)
		}
		c.MaxIssuanceWindow = conf.MaxIssuanceWindow
	}

	var submitter txbuilder.Submitter
	var gen *generator.Generator
	var remoteGenerator pb.NodeClient
	if !conf.IsGenerator {
		conn, err := rpc.NewGRPCConn(conf.GeneratorURL, conf.GeneratorAccessToken, conf.ID, conf.BlockchainID.String())
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}
		remoteGenerator = pb.NewNodeClient(conn.Conn)
		submitter = &txbuilder.RemoteGenerator{Peer: remoteGenerator}
	} else {
		gen = generator.New(c, generatorSigners, db)
		submitter = gen
	}

	// Set up the pin store for block processing
	pinStore := pin.NewStore(db)
	err = pinStore.LoadAll(ctx)
	if err != nil {
		chainlog.Fatal(ctx, chainlog.KeyError, err)
	}
	// Start listeners
	go pinStore.Listen(ctx, account.PinName, *dbURL)
	go pinStore.Listen(ctx, asset.PinName, *dbURL)

	// Setup the transaction query indexer to index every transaction.
	indexer := query.NewIndexer(db, c, pinStore)

	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	if *indexTxs {
		go pinStore.Listen(ctx, query.TxPinName, *dbURL)
		indexer.RegisterAnnotator(assets.AnnotateTxs)
		indexer.RegisterAnnotator(accounts.AnnotateTxs)
		assets.IndexAssets(indexer)
		accounts.IndexAccounts(indexer)
	}

	// GC old submitted txs periodically.
	go core.CleanupSubmittedTxs(ctx, db)

	h := &core.Handler{
		Chain:        c,
		Store:        store,
		PinStore:     pinStore,
		Assets:       assets,
		Accounts:     accounts,
		HSM:          hsm,
		Submitter:    submitter,
		TxFeeds:      &txfeed.Tracker{DB: db},
		Indexer:      indexer,
		AccessTokens: &accesstoken.CredentialStore{DB: db},
		Config:       conf,
		DB:           db,
		Addr:         *listenAddr,
		Signer:       signBlockHandler,
		AltAuth:      authLoopbackInDev,
	}
	if *rpsToken > 0 {
		h.RequestLimits = append(h.RequestLimits, core.RequestLimit{
			Key:       core.AuthID,
			Burst:     2 * (*rpsToken),
			PerSecond: *rpsToken,
		})
	}
	if *rpsRemoteAddr > 0 {
		h.RequestLimits = append(h.RequestLimits, core.RequestLimit{
			Key:       core.PeerID,
			Burst:     2 * (*rpsRemoteAddr),
			PerSecond: *rpsRemoteAddr,
		})
	}

	var (
		genhealth   = h.HealthSetter("generator")
		fetchhealth = h.HealthSetter("fetch")
	)

	go func() {
		<-c.Ready()
		height := c.Height()
		if height > 0 {
			height = height - 1
		}
		err := pinStore.CreatePin(ctx, account.PinName, height)
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}
		err = pinStore.CreatePin(ctx, asset.PinName, height)
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}
		err = pinStore.CreatePin(ctx, query.TxPinName, height)
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}
	}()

	// Note, it's important for any services that will install blockchain
	// callbacks to be initialized before leader.Run() and the http server,
	// otherwise there's a data race within protocol.Chain.
	go leader.Run(db, *listenAddr, func(ctx context.Context) {
		go h.Accounts.ExpireReservations(ctx, expireReservationsPeriod)
		if conf.IsGenerator {
			go gen.Generate(ctx, blockPeriod, genhealth)
		} else {
			go fetch.Fetch(ctx, c, remoteGenerator, fetchhealth)
		}
		go h.Accounts.ProcessBlocks(ctx)
		go h.Assets.ProcessBlocks(ctx)
		if *indexTxs {
			go h.Indexer.ProcessBlocks(ctx)
		}
	})

	return h
}

// remoteSigner defines the address and public key of another Core
// that may sign blocks produced by this generator.
type remoteSigner struct {
	client pb.SignerClient
	key    ed25519.PublicKey
	addr   string
}

func remoteSignerInfo(ctx context.Context, processID, buildTag, blockchainID string, config *config.Config) (a []*remoteSigner) {
	for _, signer := range config.Signers {
		if len(signer.Pubkey) != ed25519.PublicKeySize {
			chainlog.Fatal(ctx, chainlog.KeyError, "bad public key size", "at", "decoding signer public key")
		}
		k := ed25519.PublicKey(signer.Pubkey)
		conn, err := rpc.NewGRPCConn(signer.URL, signer.AccessToken, config.ID, config.BlockchainID.String())
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, errors.Wrap(err), "at", "connecting to signer rpc")
		}
		client := pb.NewSignerClient(conn.Conn)
		a = append(a, &remoteSigner{client: client, key: k, addr: signer.URL})
	}
	return a
}

func (s *remoteSigner) SignBlock(ctx context.Context, b *bc.Block) (signature []byte, err error) {
	// TODO(kr): We might end up serializing b multiple
	// times in multiple calls to different remoteSigners.
	// Maybe optimize that if it makes a difference.
	var buf bytes.Buffer
	_, err = b.WriteTo(&buf)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.SignBlock(ctx, &pb.SignBlockRequest{Block: buf.Bytes()})
	if err != nil {
		return nil, err
	}
	return resp.Signature, nil
}

func (s *remoteSigner) String() string {
	return s.addr
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
