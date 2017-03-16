// Command cored provides the Chain Core daemon and API server.
package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"expvar"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/kr/secureheader"

	"chain/core"
	"chain/core/blocksigner"
	"chain/core/config"
	"chain/core/generator"
	"chain/core/migrate"
	"chain/core/rpc"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/sql"
	"chain/encoding/json"
	"chain/env"
	"chain/errors"
	"chain/generated/rev"
	chainlog "chain/log"
	"chain/log/rotation"
	"chain/log/splunk"
	"chain/net/http/limit"
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
	buildTag    = "?"
	buildCommit = "?"
	buildDate   = "?"

	race          []interface{} // initialized in race.go
	httpsRedirect = true        // initialized in insecure.go

	blockPeriod              = time.Second
	expireReservationsPeriod = time.Second
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
	resetInDevIfRequested(db)

	conf, err := config.Load(ctx, db)
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
		processID += "-" + conf.ID
	}
	expvar.NewString("processID").Set(processID)

	log.SetPrefix("cored-" + buildTag + ": ")
	log.SetFlags(log.Lshortfile)
	chainlog.SetPrefix(append([]interface{}{"app", "cored", "buildtag", buildTag, "processID", processID}, race...)...)
	chainlog.SetOutput(logWriter())

	var h http.Handler
	if conf != nil {
		h = launchConfiguredCore(ctx, db, conf, processID)
	} else {
		chainlog.Printf(ctx, "Launching as unconfigured Core.")
		h = core.RunUnconfigured(ctx, db, core.AlternateAuth(authLoopbackInDev))
	}

	secureheader.DefaultConfig.PermitClearLoopback = true
	secureheader.DefaultConfig.HTTPSRedirect = httpsRedirect
	secureheader.DefaultConfig.Next = h

	// Give the remainder of this function a second to reach the
	// ListenAndServe call, then log a welcome message.
	go func() {
		time.Sleep(time.Second)
		chainlog.Printf(ctx, "Chain Core online and listening at %s", *listenAddr)
	}()

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
}

func launchConfiguredCore(ctx context.Context, db pg.DB, conf *config.Config, processID string) http.Handler {
	// Initialize the protocol.Chain.
	heights, err := txdb.ListenBlocks(ctx, *dbURL)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}
	store := txdb.NewStore(db)
	c, err := protocol.NewChain(ctx, conf.BlockchainID, store, heights)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}

	var localSigner *blocksigner.BlockSigner
	var opts []core.RunOption

	// Allow loopback/localhost requests in Developer Edition.
	opts = append(opts, core.AlternateAuth(authLoopbackInDev))
	opts = append(opts, core.IndexTransactions(*indexTxs))
	opts = append(opts, devEnableMockHSM(db)...)
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
		for _, signer := range remoteSignerInfo(ctx, processID, buildTag, conf.BlockchainID.String(), conf) {
			signers = append(signers, signer)
		}
		c.MaxIssuanceWindow = conf.MaxIssuanceWindow.Duration

		gen := generator.New(c, signers, db)
		opts = append(opts, core.GeneratorLocal(gen))
	} else {
		opts = append(opts, core.GeneratorRemote(&rpc.Client{
			BaseURL:      conf.GeneratorURL,
			AccessToken:  conf.GeneratorAccessToken,
			Username:     processID,
			CoreID:       conf.ID,
			BuildTag:     buildTag,
			BlockchainID: conf.BlockchainID.String(),
		}))
	}

	// Start up the Core. This will start up the various Core subsystems,
	// and begin leader election.
	api, err := core.Run(ctx, conf, db, *dbURL, c, store, *listenAddr, opts...)
	if err != nil {
		chainlog.Fatalkv(ctx, chainlog.KeyError, err)
	}
	return api
}

func initializeLocalSigner(ctx context.Context, conf *config.Config, db pg.DB, c *protocol.Chain, processID string) (*blocksigner.BlockSigner, error) {
	blockPub, err := hex.DecodeString(conf.BlockPub)
	if err != nil {
		return nil, err
	}

	var hsm blocksigner.Signer
	if conf.BlockHSMURL != "" {
		// TODO(ameets): potential option to take only a password when configuring
		//  and convert to an access token string here for BlockHSMAccessToken
		hsm = &remoteHSM{Client: &rpc.Client{
			BaseURL:      conf.BlockHSMURL,
			AccessToken:  conf.BlockHSMAccessToken,
			Username:     processID,
			CoreID:       conf.ID,
			BuildTag:     buildTag,
			BlockchainID: conf.BlockchainID.String(),
		}}
	} else {
		hsm, err = devHSM(db)
		if err != nil {
			return nil, err
		}
	}
	s := blocksigner.New(blockPub, hsm, db, c)
	return s, nil
}

// remoteSigner defines the address and public key of another Core
// that may sign blocks produced by this generator.
type remoteSigner struct {
	Client *rpc.Client
	Key    ed25519.PublicKey
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
		u, err := url.Parse(signer.URL)
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
			CoreID:       conf.ID,
			BuildTag:     buildTag,
			BlockchainID: blockchainID,
		}
		a = append(a, &remoteSigner{Client: client, Key: ed25519.PublicKey(signer.Pubkey)})
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
