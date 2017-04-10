// Command cored provides the Chain Core daemon and API server.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"expvar"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kr/secureheader"

	"chain/core"
	"chain/core/blocksigner"
	"chain/core/config"
	"chain/core/fileutil"
	"chain/core/generator"
	"chain/core/migrate"
	"chain/core/rpc"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/raft"
	"chain/database/sql"
	"chain/encoding/json"
	"chain/env"
	"chain/errors"
	"chain/generated/rev"
	chainlog "chain/log"
	"chain/log/rotation"
	"chain/log/splunk"
	"chain/net/http/limit"
	"chain/net/http/reqid"
	"chain/protocol"
	"chain/protocol/bc"
)

const (
	httpReadTimeout  = 2 * time.Minute
	httpWriteTimeout = time.Hour
)

var (
	// config vars
	tlsCrt        = env.String("TLSCRT", "")
	tlsKey        = env.String("TLSKEY", "")
	rootCAs       = env.String("ROOT_CA_CERTS", "") // file path
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
	dataDir       = env.String("CORED_DATA_DIR", fileutil.DefaultDir())
	bootURL       = env.String("BOOTURL", "")

	// build vars; initialized by the linker
	buildTag    = "?"
	buildCommit = "?"
	buildDate   = "?"

	race          []interface{} // initialized in race.go
	httpsRedirect = true        // initialized in plain_http.go

	// By default, requests made on the loopback interface
	// must be authenticated. To permit requests on this
	// interface use the loopback_auth build tag.
	loopbackAuth = func(req *http.Request) bool {
		return false
	}

	// By default, a core is not able to reset its data.
	// This feature can be turned on with the reset build tag.
	resetIfAllowedAndRequested = func(pg.DB) {}
)

func init() {
	var version string
	if buildTag != "?" {
		// build tag with chain-core-server- prefix indicates official release
		version = strings.TrimPrefix(buildTag, "chain-core-server-")
	} else {
		// version of the form rev123 indicates non-release build
		version = rev.ID
	}

	prodStr := "no"
	if prod {
		prodStr = "yes"
	}

	expvar.NewString("prod").Set(prodStr)
	expvar.NewString("version").Set(version)
	expvar.NewString("build_tag").Set(buildTag)
	expvar.NewString("build_date").Set(buildDate)
	expvar.NewString("build_commit").Set(buildCommit)
	expvar.NewString("runtime.GOOS").Set(runtime.GOOS)
	expvar.NewString("runtime.GOARCH").Set(runtime.GOARCH)
	expvar.NewString("runtime.Version").Set(runtime.Version())

	config.Version = version
	config.BuildCommit = buildCommit
	config.BuildDate = buildDate
	config.Production = prod
}

func main() {
	v := flag.Bool("version", false, "print version information")
	flag.Parse()

	if !*v {
		fmt.Printf("Chain Core starting...\n\n")
	}

	fmt.Printf("cored (Chain Core) %s\n", config.Version)
	fmt.Printf("production: %t\n", config.Production)
	fmt.Printf("build-commit: %v\n", config.BuildCommit)
	fmt.Printf("build-date: %v\n", config.BuildDate)
	fmt.Printf("mockhsm: %t\n", config.BuildConfig.MockHSM)
	fmt.Printf("loopback-auth: %t\n", config.BuildConfig.LoopbackAuth)
	fmt.Printf("protected-db: %t\n", config.BuildConfig.ProtectedDB)
	fmt.Printf("reset: %t\n", config.BuildConfig.Reset)

	if *v {
		return
	}

	fmt.Printf("\n")
	runServer()
}

func runServer() {
	maybeMonitorIfOnWindows() // special-case windows

	ctx := context.Background()
	env.Parse()

	raftDir := filepath.Join(*dataDir, "raft") // TODO(kr): better name for this
	// TODO(tessr): remove tls param once we have tls everywhere
	raftDB, err := raft.Start(*listenAddr, raftDir, *bootURL, *tlsCrt != "")
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}

	// We add handlers to our serve mux in two phases. In the first phase, we start
	// listening on the raft routes (`/raft`). This allows us to do things like
	// read the config value stored in raft storage. (A new node in a raft cluster
	// can't read values without kicking off a consensus round, which in turn
	// requires this node to be listening for raft requests.)
	//
	// Once this node is able to read the config value, it can set up the remaining
	// cored functionality, and add the rest of the core routes to the serve mux.
	// That is the second phase.
	mux := http.NewServeMux()
	mux.Handle("/raft/", raftDB)

	var handler http.Handler = mux
	handler = reqid.Handler(handler)

	secureheader.DefaultConfig.PermitClearLoopback = true
	secureheader.DefaultConfig.HTTPSRedirect = httpsRedirect
	secureheader.DefaultConfig.Next = handler

	server := &http.Server{
		Addr:         *listenAddr,
		Handler:      secureheader.DefaultConfig,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		// Disable HTTP/2 for now until the Go implementation is more stable.
		// https://github.com/golang/go/issues/16450
		// https://github.com/golang/go/issues/17071
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}

	// The `ListenAndServe` call has to happen in its own goroutine because
	// it's blocking and we need to proceed to the rest of the core setup after
	// we call it.
	go func() {
		if *tlsCrt != "" {
			cert, err := tls.X509KeyPair([]byte(*tlsCrt), []byte(*tlsKey))
			if err != nil {
				chainlog.Fatalkv(ctx, chainlog.KeyError, errors.Wrap(err, "parsing tls X509 key pair"))
			}

			server.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
			err = server.ListenAndServeTLS("", "") // uses TLS certs from above
			if err != nil {
				chainlog.Fatalkv(ctx, chainlog.KeyError, errors.Wrap(err, "ListenAndServeTLS"))
			}
		} else {
			err = server.ListenAndServe()
			if err != nil {
				chainlog.Fatalkv(ctx, chainlog.KeyError, errors.Wrap(err, "ListenAndServe"))
			}
		}
	}()

	if *rootCAs != "" {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
			RootCAs: loadRootCAs(*rootCAs),
		}
	}

	sql.EnableQueryLogging(*logQueries)
	db, err := sql.Open("hapg", *dbURL)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}
	db.SetMaxOpenConns(*maxDBConns)
	db.SetMaxIdleConns(*maxDBConns)

	err = migrate.Run(db)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}
	resetIfAllowedAndRequested(db, raftDB)

	conf, err := config.Load(ctx, db, raftDB)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}

	// Initialize internode rpc clients.
	hostname, err := os.Hostname()
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}
	processID := fmt.Sprintf("chain-%s-%d", hostname, os.Getpid())
	if conf != nil {
		processID += "-" + conf.Id
	}
	expvar.NewString("processID").Set(processID)

	log.SetPrefix("cored-" + buildTag + ": ")
	log.SetFlags(log.Lshortfile)
	chainlog.SetPrefix(append([]interface{}{"app", "cored", "buildtag", buildTag, "processID", processID}, race...)...)
	chainlog.SetOutput(logWriter())

	var h http.Handler
	if conf != nil {
		h = launchConfiguredCore(ctx, raftDB, db, conf, processID)
	} else {
		chainlog.Printf(ctx, "Launching as unconfigured Core.")
		h = core.RunUnconfigured(ctx, db, raftDB, core.AlternateAuth(loopbackAuth))
	}
	mux.Handle("/", h)
	chainlog.Printf(ctx, "Chain Core online and listening at %s", *listenAddr)

	// block forever without using any resources so this process won't quit while
	// the goroutine containing ListenAndServe is still working
	select {}
}

func launchConfiguredCore(ctx context.Context, raftDB *raft.Service, db *sql.DB, conf *config.Config, processID string) http.Handler {
	// Initialize the protocol.Chain.
	heights, err := txdb.ListenBlocks(ctx, *dbURL)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}
	store := txdb.NewStore(db)
	c, err := protocol.NewChain(ctx, *conf.BlockchainId, store, heights)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}

	var localSigner *blocksigner.BlockSigner
	var opts []core.RunOption

	// Allow loopback/localhost requests in Developer Edition.
	opts = append(opts, core.AlternateAuth(loopbackAuth))
	opts = append(opts, core.IndexTransactions(*indexTxs))
	opts = append(opts, enableMockHSM(db)...)
	// Add any configured API request rate limits.
	if *rpsToken > 0 {
		opts = append(opts, core.RateLimit(limit.AuthUserID, 2*(*rpsToken), *rpsToken))
	}
	if *rpsRemoteAddr > 0 {
		opts = append(opts, core.RateLimit(limit.RemoteAddrID, 2*(*rpsRemoteAddr), *rpsRemoteAddr))
	}
	// If the Core is configured as a block signer, add the sign-block RPC handler.
	if conf.IsSigner {
		localSigner, err = initializeLocalSigner(ctx, conf, db, c, processID)
		if err != nil {
			chainlog.Fatalkv(ctx, chainlog.KeyError, err)
		}
		opts = append(opts, core.BlockSigner(localSigner.ValidateAndSignBlock))
	}

	// The Core is either configured as a generator or not. If it's configured
	// as a generator, instantiate the generator with the configured local and
	// remote block signers. Provide a launch option to the Core to use the
	// generator.
	//
	// If the Core is not a generator, provide an RPC client for the generator
	// so that the Core can replicate blocks.
	if conf.IsGenerator {
		var signers []generator.BlockSigner
		if localSigner != nil {
			signers = append(signers, localSigner)
		}
		for _, signer := range remoteSignerInfo(ctx, processID, buildTag, conf.BlockchainId.String(), conf) {
			signers = append(signers, signer)
		}
		c.MaxIssuanceWindow = bc.MillisDuration(conf.MaxIssuanceWindowMs)

		gen := generator.New(c, signers, db)
		opts = append(opts, core.GeneratorLocal(gen))
	} else {
		opts = append(opts, core.GeneratorRemote(&rpc.Client{
			BaseURL:      conf.GeneratorUrl,
			AccessToken:  conf.GeneratorAccessToken,
			Username:     processID,
			CoreID:       conf.Id,
			BuildTag:     buildTag,
			BlockchainID: conf.BlockchainId.String(),
		}))
	}

	// Start up the Core. This will start up the various Core subsystems,
	// and begin leader election.
	api, err := core.Run(ctx, conf, db, *dbURL, raftDB, c, store, *listenAddr, opts...)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}
	return api
}

func initializeLocalSigner(ctx context.Context, conf *config.Config, db pg.DB, c *protocol.Chain, processID string) (*blocksigner.BlockSigner, error) {
	var hsm blocksigner.Signer
	if conf.BlockHsmUrl != "" {
		// TODO(ameets): potential option to take only a password when configuring
		//  and convert to an access token string here for BlockHSMAccessToken
		hsm = &remoteHSM{Client: &rpc.Client{
			BaseURL:      conf.BlockHsmUrl,
			AccessToken:  conf.BlockHsmAccessToken,
			Username:     processID,
			CoreID:       conf.Id,
			BuildTag:     buildTag,
			BlockchainID: conf.BlockchainId.String(),
		}}
	} else {
		var err error
		hsm, err = mockHSM(db)
		if err != nil {
			return nil, err
		}
	}
	blockPub := ed25519.PublicKey(conf.BlockPub)
	s := blocksigner.New(blockPub, hsm, db, c)
	return s, nil
}

// remoteHSM is a client wrapper for an hsm that is used as a blocksigner.Signer
type remoteHSM struct {
	Client *rpc.Client
}

func (h *remoteHSM) Sign(ctx context.Context, pk ed25519.PublicKey, bh *bc.BlockHeader) (signature []byte, err error) {
	body := struct {
		Block *bc.BlockHeader `json:"block"`
		Pub   json.HexBytes   `json:"pubkey"`
	}{bh, json.HexBytes(pk[:])}
	err = h.Client.Call(ctx, "/sign-block", body, &signature)
	return
}

func remoteSignerInfo(ctx context.Context, processID, buildTag, blockchainID string, conf *config.Config) (a []*remoteSigner) {
	for _, signer := range conf.Signers {
		u, err := url.Parse(signer.Url)
		if err != nil {
			chainlog.Fatalkv(ctx, chainlog.KeyError, err)
		}
		if len(signer.Pubkey) != ed25519.PublicKeySize {
			chainlog.Fatalkv(ctx, chainlog.KeyError, errors.Wrap(err), "at", "decoding signer public key")
		}
		client := &rpc.Client{
			BaseURL:      u.String(),
			AccessToken:  signer.AccessToken,
			Username:     processID,
			CoreID:       conf.Id,
			BuildTag:     buildTag,
			BlockchainID: blockchainID,
		}
		a = append(a, &remoteSigner{Client: client, Key: ed25519.PublicKey(signer.Pubkey)})
	}
	return a
}

// remoteSigner defines the address and public key of another Core
// that may sign blocks produced by this generator.
type remoteSigner struct {
	Client *rpc.Client
	Key    ed25519.PublicKey
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

// loadRootCAs reads a list of PEM-encoded X.509 certificates from name
func loadRootCAs(name string) *x509.CertPool {
	pem, err := ioutil.ReadFile(name)
	if err != nil {
		chainlog.Fatalkv(context.Background(), chainlog.KeyError, err)
	}
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(pem)
	if !ok {
		chainlog.Fatalkv(context.Background(), chainlog.KeyError, "no certs found in "+name)
	}
	return pool
}
