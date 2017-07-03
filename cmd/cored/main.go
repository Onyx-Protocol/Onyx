// Command cored provides the Chain Core daemon and API server.
package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kr/secureheader"

	"chain/core"
	"chain/core/accesstoken"
	"chain/core/blocksigner"
	"chain/core/config"
	"chain/core/generator"
	"chain/core/migrate"
	"chain/core/rpc"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/sinkdb"
	"chain/database/sqlutil"
	"chain/encoding/json"
	"chain/env"
	"chain/errors"
	"chain/generated/rev"
	chainlog "chain/log"
	"chain/log/rotation"
	"chain/log/splunk"
	"chain/net/http/authz"
	"chain/net/http/limit"
	"chain/net/http/reqid"
	"chain/net/raft"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
)

const (
	httpReadTimeout  = 2 * time.Minute
	httpWriteTimeout = time.Hour
)

var (
	// config vars
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
	home          = config.HomeDirFromEnvironment()

	version string // initialized in init()

	// build vars; initialized by the linker
	buildTag    = "?"
	buildCommit = "?"
	buildDate   = "?"

	race []interface{} // initialized in race.go

	// By default, a core is not able to reset its data.
	// This feature can be turned on with the reset build tag.
	resetIfAllowedAndRequested = func(pg.DB, *sinkdb.DB) {}

	// See localhost_auth.go.
	builtinGrants []*authz.Grant
)

func init() {
	if buildTag != "?" {
		// build tag with chain-core-server- prefix indicates official release
		version = strings.TrimPrefix(buildTag, "chain-core-server-")
	} else {
		// version of the form rev123 indicates non-release build
		version = rev.ID
	}

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
}

func main() {
	v := flag.Bool("version", false, "print version information")
	flag.Parse()

	if !*v {
		fmt.Printf("Chain Core starting...\n\n")
	}

	fmt.Printf("cored (Chain Core) %s\n", config.Version)
	fmt.Printf("build-commit: %v\n", config.BuildCommit)
	fmt.Printf("build-date: %v\n", config.BuildDate)
	fmt.Printf("mockhsm: %t\n", config.BuildConfig.MockHSM)
	fmt.Printf("localhost_auth: %t\n", config.BuildConfig.LocalhostAuth)
	fmt.Printf("reset: %t\n", config.BuildConfig.Reset)
	fmt.Printf("http_ok: %t\n", config.BuildConfig.HTTPOk)
	fmt.Printf("init_cluster: %t\n", config.BuildConfig.InitCluster)

	if *v {
		return
	}

	fmt.Printf("\n")

	maybeMonitorIfOnWindows() // special-case windows

	ctx := context.Background()
	env.Parse()
	warnCompat(ctx)

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}
	listener, tlsConfig, err := maybeUseTLS(listener)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}

	// TODO(kr): make core.UseTLS take just an http client
	// and use this object in it.
	httpClient := new(http.Client)
	httpClient.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,

		// The following fields are default values
		// copied from DefaultTransport.
		// (When you change them, be sure to move them
		// above this line so this comment stays true.)
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	raftDir := filepath.Join(home, "raft") // TODO(kr): better name for this
	sdb, err := sinkdb.Open(*listenAddr, raftDir, httpClient)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}

	// In Developer Edition, automatically create a new cluster if
	// there's no existing raft cluster.
	if config.BuildConfig.InitCluster {
		err = sdb.RaftService().Init()
		if err != nil && errors.Root(err) != raft.ErrExistingCluster {
			chainlog.Fatalkv(ctx, chainlog.KeyError, err)
		}
	}

	driver := pg.NewDriver()
	if *logQueries {
		driver = sqlutil.LogDriver(driver)
	}
	sql.Register("coredpg", driver)
	db, err := sql.Open("coredpg", *dbURL)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}
	db.SetMaxOpenConns(*maxDBConns)
	db.SetMaxIdleConns(*maxDBConns)

	err = migrate.Run(db)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}

	accessTokens := &accesstoken.CredentialStore{DB: db}

	// We add handlers to our serve mux in two phases. In the first phase, we start
	// listening on the raft routes (`/raft`). This allows us to do things like
	// read the config value stored in raft storage. (A new node in a raft cluster
	// can't read values without kicking off a consensus round, which in turn
	// requires this node to be listening for raft requests.)
	//
	// Once this node is able to read the config value, it can set up the remaining
	// cored functionality, and add the rest of the core routes to the serve mux.
	// That is the second phase.
	//
	// The waitHandler accepts incoming requests, but blocks until its underlying
	// handler is set, when the second phase is complete.
	var coreHandler waitHandler
	coreHandler.wg.Add(1)
	mux := http.NewServeMux()
	mux.Handle("/raft/", sdb.RaftService())
	mux.Handle("/", &coreHandler)

	var handler http.Handler = mux
	handler = core.AuthHandler(handler, sdb, accessTokens, tlsConfig, builtinGrants)
	handler = core.RedirectHandler(handler)
	handler = reqid.Handler(handler)

	secureheader.DefaultConfig.PermitClearLoopback = true
	secureheader.DefaultConfig.HTTPSRedirect = false
	secureheader.DefaultConfig.Next = handler

	server := &http.Server{
		// Note: we should not set TLSConfig here;
		// we took care of TLS with the listener in maybeUseTLS.
		Handler:      secureheader.DefaultConfig,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		// Disable HTTP/2 for now until the Go implementation is more stable.
		// https://github.com/golang/go/issues/16450
		// https://github.com/golang/go/issues/17071
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}

	// The `Serve` call has to happen in its own goroutine because
	// it's blocking and we need to proceed to the rest of the core setup after
	// we call it.
	go func() {
		err := server.Serve(listener)
		chainlog.Fatalkv(ctx, chainlog.KeyError, errors.Wrap(err, "Serve"))
	}()

	// Verify that we're connected to the rest of the cluster, if initialized.
	err = errors.Root(sdb.Ping())
	if err == context.DeadlineExceeded {
		chainlog.Fatalkv(ctx, chainlog.KeyError, "Unable to reach rest of raft cluster. Was the node evicted?")
	} else if err != nil && err != raft.ErrUninitialized {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}

	resetIfAllowedAndRequested(db, sdb)

	conf, err := config.Load(ctx, db, sdb)
	if err != nil && errors.Root(err) != raft.ErrUninitialized {
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

	log.SetPrefix("cored-" + version + ": ")
	log.SetFlags(log.Lshortfile)
	chainlog.SetPrefix(append([]interface{}{"app", "cored", "version", version, "processID", processID}, race...)...)
	chainlog.SetOutput(logWriter())

	var h http.Handler
	if conf != nil {
		h = launchConfiguredCore(ctx, sdb, db, conf, processID, httpClient, core.UseTLS(tlsConfig))
	} else {
		var opts []core.RunOption
		opts = append(opts, core.UseTLS(tlsConfig))
		opts = append(opts, enableMockHSM(db)...)
		chainlog.Printf(ctx, "Launching as unconfigured Core.")
		h = core.RunUnconfigured(ctx, db, sdb, *listenAddr, opts...)

		go func() {
			for {
				core.CheckConfigMaybeExec(ctx, sdb, *listenAddr)
				time.Sleep(5 * time.Second)
			}
		}()
	}
	coreHandler.Set(h)
	chainlog.Printf(ctx, "Chain Core online and listening at %s", *listenAddr)

	// block forever without using any resources so this process won't quit while
	// the goroutine containing ListenAndServe is still working
	select {}
}

// maybeUseTLS loads the TLS cert and key (if so configured)
// and wraps ln in a TLS listener. If using TLS the config
// will be returned. Otherwise the second return arg will
// be nil.
func maybeUseTLS(ln net.Listener) (net.Listener, *tls.Config, error) {
	c, err := core.TLSConfig(
		filepath.Join(home, "tls.crt"),
		filepath.Join(home, "tls.key"),
		*rootCAs,
	)
	if err == core.ErrNoTLS && config.BuildConfig.HTTPOk {
		return ln, nil, nil // files & env vars don't exist; don't want TLS
	} else if err != nil {
		return nil, nil, err
	}
	ln = tls.NewListener(ln, c)
	return ln, c, nil
}

func launchConfiguredCore(ctx context.Context, sdb *sinkdb.DB, db *sql.DB, conf *config.Config, processID string, httpClient *http.Client, opts ...core.RunOption) http.Handler {
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
		localSigner, err = initializeLocalSigner(ctx, conf, db, c, processID, httpClient)
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
		for _, signer := range remoteSignerInfo(ctx, processID, conf.BlockchainId.String(), conf, httpClient) {
			signers = append(signers, signer)
		}
		c.MaxIssuanceWindow = bc.MillisDuration(conf.MaxIssuanceWindowMs)

		gen := generator.New(c, signers, db)
		opts = append(opts, core.GeneratorLocal(gen))
	} else {
		opts = append(opts, core.GeneratorRemote(&rpc.Client{
			BaseURL:      conf.GeneratorUrl,
			AccessToken:  conf.GeneratorAccessToken,
			ProcessID:    processID,
			CoreID:       conf.Id,
			Version:      version,
			BlockchainID: conf.BlockchainId.String(),
			Client:       httpClient,
		}))
	}

	// Start up the Core. This will start up the various Core subsystems,
	// and begin leader election.
	api, err := core.Run(ctx, conf, db, *dbURL, sdb, c, store, *listenAddr, opts...)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}
	return api
}

func initializeLocalSigner(ctx context.Context, conf *config.Config, db pg.DB, c *protocol.Chain, processID string, httpClient *http.Client) (*blocksigner.BlockSigner, error) {
	var hsm blocksigner.Signer
	if conf.BlockHsmUrl != "" {
		// TODO(ameets): potential option to take only a password when configuring
		//  and convert to an access token string here for BlockHSMAccessToken
		hsm = &remoteHSM{Client: &rpc.Client{
			BaseURL:      conf.BlockHsmUrl,
			AccessToken:  conf.BlockHsmAccessToken,
			ProcessID:    processID,
			CoreID:       conf.Id,
			Version:      version,
			BlockchainID: conf.BlockchainId.String(),
			Client:       httpClient,
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

func (h *remoteHSM) Sign(ctx context.Context, pk ed25519.PublicKey, bh *legacy.BlockHeader) (signature []byte, err error) {
	body := struct {
		Block *legacy.BlockHeader `json:"block"`
		Pub   json.HexBytes       `json:"pubkey"`
	}{bh, json.HexBytes(pk[:])}
	err = h.Client.Call(ctx, "/sign-block", body, &signature)
	return
}

func remoteSignerInfo(ctx context.Context, processID, blockchainID string, conf *config.Config, httpClient *http.Client) (a []*remoteSigner) {
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
			ProcessID:    processID,
			CoreID:       conf.Id,
			Version:      version,
			BlockchainID: blockchainID,
			Client:       httpClient,
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

func (s *remoteSigner) SignBlock(ctx context.Context, marshalledBlock []byte) (signature []byte, err error) {
	err = s.Client.Call(ctx, "/rpc/signer/sign-block", string(marshalledBlock), &signature)
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

type waitHandler struct {
	h  http.Handler
	wg sync.WaitGroup
}

func (wh *waitHandler) Set(h http.Handler) {
	wh.h = h
	wh.wg.Done()
}

func (wh *waitHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	wh.wg.Wait()
	wh.h.ServeHTTP(w, req)
}
