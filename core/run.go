package core

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"time"

	"chain/core/accesstoken"
	"chain/core/account"
	"chain/core/asset"
	"chain/core/config"
	"chain/core/fetch"
	"chain/core/generator"
	"chain/core/leader"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/rpc"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/core/txfeed"
	"chain/database/pg"
	"chain/database/sinkdb"
	"chain/log"
	"chain/net/http/authz"
	"chain/protocol"
	"chain/protocol/bc/legacy"
)

const (
	blockPeriod              = time.Second
	expireReservationsPeriod = time.Second
)

// RunOption describes a runtime configuration option.
type RunOption func(*API)

// UseTLS configures the Core to use TLS with the given config
// when communicating between Core processes.
// If c is nil, TLS is disabled.
func UseTLS(c *tls.Config) RunOption {
	return func(a *API) {
		a.httpClient = new(http.Client)
		a.httpClient.Transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSClientConfig:       c,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		if c != nil {
			// TODO(kr): set Leaf in TLSConfig and use that here.
			x509Cert, err := x509.ParseCertificate(c.Certificates[0].Certificate[0])
			if err != nil {
				panic(err)
			}
			a.internalSubj = x509Cert.Subject
		}
	}
}

// BlockSigner configures the Core to use signFn to handle block-signing
// requests. In production, this will be a function to call out to signerd
// and its HSM. In development, it'll use the MockHSM.
func BlockSigner(signFn func(context.Context, *legacy.Block) ([]byte, error)) RunOption {
	return func(a *API) { a.signer = signFn }
}

// GeneratorLocal configures the launched Core to run as a Generator.
func GeneratorLocal(gen *generator.Generator) RunOption {
	return func(a *API) {
		if a.remoteGenerator != nil {
			panic("core configured with local and remote generator")
		}
		a.generator = gen
		a.submitter = gen
	}
}

// GeneratorRemote configures the launched Core to fetch blocks from
// the provided remote generator.
func GeneratorRemote(client *rpc.Client) RunOption {
	return func(a *API) {
		if a.generator != nil {
			panic("core configured with local and remote generator")
		}
		a.remoteGenerator = client
		a.submitter = &txbuilder.RemoteGenerator{Peer: client}
	}
}

// IndexTransactions configures whether or not transactions should be
// annotated and indexed for the query engine.
func IndexTransactions(b bool) RunOption {
	return func(a *API) { a.indexTxs = b }
}

// RateLimit adds a rate-limiting restriction, using keyFn to extract the
// key to rate limit on. It will allow up to burst requests in the bucket
// and will refill the bucket at perSecond tokens per second.
func RateLimit(keyFn func(*http.Request) string, burst, perSecond int) RunOption {
	return func(a *API) {
		a.requestLimits = append(a.requestLimits, requestLimit{
			key:       keyFn,
			burst:     burst,
			perSecond: perSecond,
		})
	}
}

// RunUnconfigured launches a new unconfigured Chain Core. This is
// used for Chain Core Developer Edition to expose the configuration UI
// in the dashboard. API authentication still applies to an unconfigured
// Chain Core.
func RunUnconfigured(ctx context.Context, db pg.DB, sdb *sinkdb.DB, routableAddress string, opts ...RunOption) *API {
	a := &API{
		db:           db,
		sdb:          sdb,
		accessTokens: &accesstoken.CredentialStore{DB: db},
		grants:       authz.NewStore(sdb, GrantPrefix),
		options:      config.New(sdb),
		mux:          http.NewServeMux(),
		addr:         routableAddress,
	}
	for _, opt := range opts {
		opt(a)
	}

	// Construct the complete http.Handler once.
	a.buildHandler()
	return a
}

// Run launches a new configured Chain Core. It will start goroutines
// for the various Core subsystems and enter leader election. It will not
// start listening for HTTP requests. To begin serving HTTP requests, use
// API.Handler to retrieve an http.Handler that can be used in a call to
// http.ListenAndServe.
//
// Either the GeneratorLocal or the GeneratorRemote RunOption is
// required.
func Run(
	ctx context.Context,
	conf *config.Config,
	db pg.DB,
	dbURL string,
	sdb *sinkdb.DB,
	c *protocol.Chain,
	store *txdb.Store,
	routableAddress string,
	opts ...RunOption,
) (*API, error) {
	// Set up the pin store for block processing
	pinStore := pin.NewStore(db)
	err := pinStore.LoadAll(ctx)
	if err != nil {
		return nil, err
	}
	// Start listeners
	go pinStore.Listen(ctx, account.PinName, dbURL)
	go pinStore.Listen(ctx, account.ExpirePinName, dbURL)
	go pinStore.Listen(ctx, account.DeleteSpentsPinName, dbURL)
	go pinStore.Listen(ctx, asset.PinName, dbURL)

	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	indexer := query.NewIndexer(db, c, pinStore)

	a := &API{
		chain:        c,
		store:        store,
		pinStore:     pinStore,
		assets:       assets,
		accounts:     accounts,
		txFeeds:      &txfeed.Tracker{DB: db},
		indexer:      indexer,
		accessTokens: &accesstoken.CredentialStore{DB: db},
		grants:       authz.NewStore(sdb, GrantPrefix),
		config:       conf,
		options:      config.New(sdb),
		db:           db,
		sdb:          sdb,
		mux:          http.NewServeMux(),
		addr:         routableAddress,
	}
	for _, opt := range opts {
		opt(a)
	}
	if a.remoteGenerator == nil && a.generator == nil {
		return nil, errors.New("no generator configured")
	}

	if a.indexTxs {
		go pinStore.Listen(ctx, query.TxPinName, dbURL)
		a.indexer.RegisterAnnotator(a.assets.AnnotateTxs)
		a.indexer.RegisterAnnotator(a.accounts.AnnotateTxs)
		a.assets.IndexAssets(a.indexer)
		a.accounts.IndexAccounts(a.indexer)
	}

	// Clean up expired UTXO reservations periodically.
	go accounts.ExpireReservations(ctx, expireReservationsPeriod)

	// GC old submitted txs periodically.
	go cleanUpSubmittedTxs(ctx, a.db)

	// When this cored becomes leader, run a.lead to perform
	// leader-only Core duties.
	a.leader = leader.Run(ctx, db, routableAddress, a.lead)

	// Construct the complete http.Handler once.
	a.buildHandler()

	return a, nil
}

// lead is called by the core/leader package when this cored instance
// becomes leader of the Core.
func (a *API) lead(ctx context.Context) {
	if !a.config.IsGenerator {
		fetch.Init(ctx, a.remoteGenerator)
		// If don't have any blocks, bootstrap from the generator's
		// latest snapshot.
		if a.chain.Height() == 0 {
			sp := fetch.BootstrapSnapshot(ctx, a.chain, a.store, a.remoteGenerator, a.healthSetter("fetch"))

			// Save the downloading snapshot to the api so that /info can
			// return its current status.
			a.downloadingSnapshotMu.Lock()
			a.downloadingSnapshot = sp
			a.downloadingSnapshotMu.Unlock()
			// Wait for the snapshot download to finish before continuing.
			sp.Wait()
		}
	}

	// This process just became leader, so it's responsible
	// for recovering after the previous leader's exit.
	_, _, err := a.chain.Recover(ctx)
	if err != nil {
		log.Fatalkv(ctx, log.KeyError, err)
	}

	// Create all of the block processor pins if they don't already exist.
	pinHeight := a.chain.Height()
	if pinHeight > 0 {
		pinHeight = pinHeight - 1
	}
	pins := []string{account.PinName, account.ExpirePinName, account.DeleteSpentsPinName, asset.PinName, query.TxPinName}
	for _, p := range pins {
		err = a.pinStore.CreatePin(ctx, p, pinHeight)
		if err != nil {
			log.Fatalkv(ctx, log.KeyError, err)
		}
	}

	if a.config.IsGenerator {
		go a.generator.Generate(ctx, blockPeriod, a.healthSetter("generator"))
	} else {
		// Remove the downloading snapshot if there was one. The core
		// has recovered and will now start syncing blocks.
		a.downloadingSnapshotMu.Lock()
		a.downloadingSnapshot = nil
		a.downloadingSnapshotMu.Unlock()

		go fetch.Fetch(ctx, a.chain, a.remoteGenerator, a.healthSetter("fetch"))
	}
	go a.accounts.ProcessBlocks(ctx)
	go a.assets.ProcessBlocks(ctx)
	if a.indexTxs {
		go a.indexer.ProcessBlocks(ctx)
	}
}
