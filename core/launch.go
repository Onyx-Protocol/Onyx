package core

import (
	"context"
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
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
)

const (
	blockPeriod              = time.Second
	expireReservationsPeriod = time.Second
)

// LaunchOption describes a runtime configuration option.
type LaunchOption func(*API)

// AlternateAuth configures the Core to use authFn to authenticate
// incoming requests in addition to the default access token authentication.
func AlternateAuth(authFn func(*http.Request) bool) LaunchOption {
	return func(a *API) { a.altAuth = authFn }
}

// BlockSigner configures the Core to use signFn to handle block-signing
// requests. In production, this will be a function to call out to signerd
// and its HSM. In development, it'll use the MockHSM.
func BlockSigner(signFn func(context.Context, *bc.Block) ([]byte, error)) LaunchOption {
	return func(a *API) { a.signer = signFn }
}

// IndexTransactions configures whether or not transactions should be
// annotated and indexed for the query engine.
func IndexTransactions(b bool) LaunchOption {
	return func(a *API) { a.indexTxs = b }
}

// RateLimit adds a rate-limiting restriction, using keyFn to extract the
// key to rate limit on. It will allow up to burst requests in the bucket
// and will refill the bucket at perSecond tokens per second.
func RateLimit(keyFn func(*http.Request) string, burst, perSecond int) LaunchOption {
	return func(a *API) {
		a.requestLimits = append(a.requestLimits, requestLimit{
			key:       keyFn,
			burst:     burst,
			perSecond: perSecond,
		})
	}
}

// LocalGenerator configures the launched Core to run as a Generator.
func LocalGenerator(gen *generator.Generator) LaunchOption {
	return func(a *API) {
		if a.remoteGenerator != nil {
			panic("core configured with local and remote generator")
		}
		a.generator = gen
		a.submitter = gen
	}
}

// RemoteGenerator configures the launched Core to fetch blocks from
// the provided remote generator.
func RemoteGenerator(client *rpc.Client) LaunchOption {
	return func(a *API) {
		if a.generator != nil {
			panic("core configured with local and remote generator")
		}
		a.remoteGenerator = client
		a.submitter = &txbuilder.RemoteGenerator{Peer: client}
	}
}

// LaunchUnconfigured launches a new unconfigured Chain Core. This is
// used for Chain Core Developer Edition to expose the configuration UI
// in the dashboard. API authentication still applies to an unconfigured
// Chain Core.
func LaunchUnconfigured(ctx context.Context, db pg.DB, opts ...LaunchOption) *API {
	a := &API{
		db:           db,
		accessTokens: &accesstoken.CredentialStore{DB: db},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Launch launches a new configured Chain Core. It will start goroutines
// for the various Core subsystems and enter leader election. It will not
// start listening for HTTP requests. To begin serving HTTP requests, use
// API.Handler to retrieve an http.Handler that can be used in a call to
// http.ListenAndServe.
func Launch(
	ctx context.Context,
	conf *config.Config,
	db pg.DB,
	dbURL string,
	c *protocol.Chain,
	store *txdb.Store,
	routableAddress string,
	opts ...LaunchOption,
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
		config:       conf,
		db:           db,
		addr:         routableAddress,
	}
	for _, opt := range opts {
		opt(a)
	}
	a.fetchhealth = a.HealthSetter("fetch")
	a.genhealth = a.HealthSetter("generator")

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
	go leader.Run(db, routableAddress, a.lead)

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
			fetch.BootstrapSnapshot(ctx, a.chain, a.store, a.remoteGenerator, a.fetchhealth)
		}
	}

	// This process just became leader, so it's responsible
	// for recovering after the previous leader's exit.
	recoveredBlock, recoveredSnapshot, err := a.chain.Recover(ctx)
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
		go a.generator.Generate(ctx, blockPeriod, a.genhealth, recoveredBlock, recoveredSnapshot)
	} else {
		go fetch.Fetch(ctx, a.chain, a.remoteGenerator, a.fetchhealth, recoveredBlock, recoveredSnapshot)
	}
	go a.accounts.ProcessBlocks(ctx)
	go a.assets.ProcessBlocks(ctx)
	if a.indexTxs {
		go a.indexer.ProcessBlocks(ctx)
	}
}
