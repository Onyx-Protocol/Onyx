// Package core implements Chain Core and its API.
package core

import (
	"context"
	"crypto/x509/pkix"
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"
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
	"chain/database/raft"
	"chain/encoding/json"
	"chain/errors"
	"chain/generated/dashboard"
	"chain/net/http/authn"
	"chain/net/http/authz"
	"chain/net/http/gzip"
	"chain/net/http/httpjson"
	"chain/net/http/limit"
	"chain/net/http/static"
	"chain/protocol"
	"chain/protocol/bc/legacy"
)

const (
	defGenericPageSize = 100
)

// TODO(kr): change this to "network" or something.
const networkRPCPrefix = "/rpc/"

var (
	errNotFound         = errors.New("not found")
	errRateLimited      = errors.New("request limit exceeded")
	errLeaderElection   = errors.New("no leader; pending election")
	errNotAuthenticated = errors.New("not authenticated")
)

// API serves the Chain HTTP API
type API struct {
	chain           *protocol.Chain
	store           *txdb.Store
	pinStore        *pin.Store
	assets          *asset.Registry
	accounts        *account.Manager
	indexer         *query.Indexer
	txFeeds         *txfeed.Tracker
	accessTokens    *accesstoken.CredentialStore
	config          *config.Config
	submitter       txbuilder.Submitter
	db              pg.DB
	raftDB          *raft.Service
	mux             *http.ServeMux
	handler         http.Handler
	leader          leaderProcess
	addr            string
	signer          func(context.Context, *legacy.Block) ([]byte, error)
	requestLimits   []requestLimit
	generator       *generator.Generator
	remoteGenerator *rpc.Client
	indexTxs        bool
	forwardUsingTLS bool
	internalSubj    pkix.Name
	httpClient      *http.Client

	downloadingSnapshotMu sync.Mutex
	downloadingSnapshot   *fetch.SnapshotProgress

	healthMu     sync.Mutex
	healthErrors map[string]interface{}
}

func (a *API) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	a.handler.ServeHTTP(rw, req)
}

type leaderProcess interface {
	State() leader.ProcessState
	Address(context.Context) (string, error)
}

type requestLimit struct {
	key       func(*http.Request) string
	burst     int
	perSecond int
}

func maxBytes(h http.Handler) http.Handler {
	const maxReqSize = 1e7 // 10MB
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// A block can easily be bigger than maxReqSize, but everything
		// else should be pretty small.
		if req.URL.Path != networkRPCPrefix+"signer/sign-block" {
			req.Body = http.MaxBytesReader(w, req.Body, maxReqSize)
		}
		h.ServeHTTP(w, req)
	})
}

func (a *API) needConfig() func(f interface{}) http.Handler {
	if a.config == nil {
		return func(f interface{}) http.Handler {
			return alwaysError(errUnconfigured)
		}
	}
	return jsonHandler
}

// buildHandler adds the Core API routes to a preexisting http handler.
func (a *API) buildHandler() {
	needConfig := a.needConfig()

	resetAllowed := func(h http.Handler) http.Handler { return alwaysError(errNoReset) }
	if config.BuildConfig.Reset {
		resetAllowed = func(h http.Handler) http.Handler { return h }
	}

	m := a.mux
	m.Handle("/", alwaysError(errNotFound))

	m.Handle("/create-account", needConfig(a.createAccount))
	m.Handle("/create-asset", needConfig(a.createAsset))
	m.Handle("/update-account-tags", needConfig(a.updateAccountTags))
	m.Handle("/update-asset-tags", needConfig(a.updateAssetTags))
	m.Handle("/build-transaction", needConfig(a.build))
	m.Handle("/submit-transaction", needConfig(a.submit))
	m.Handle("/create-control-program", needConfig(a.createControlProgram)) // DEPRECATED
	m.Handle("/create-account-receiver", needConfig(a.createAccountReceiver))
	m.Handle("/create-transaction-feed", needConfig(a.createTxFeed))
	m.Handle("/get-transaction-feed", needConfig(a.getTxFeed))
	m.Handle("/update-transaction-feed", needConfig(a.updateTxFeed))
	m.Handle("/delete-transaction-feed", needConfig(a.deleteTxFeed))
	m.Handle("/mockhsm", alwaysError(errNoMockHSM))
	m.Handle("/list-accounts", needConfig(a.listAccounts))
	m.Handle("/list-assets", needConfig(a.listAssets))
	m.Handle("/list-transaction-feeds", needConfig(a.listTxFeeds))
	m.Handle("/list-transactions", needConfig(a.listTransactions))
	m.Handle("/list-balances", needConfig(a.listBalances))
	m.Handle("/list-unspent-outputs", needConfig(a.listUnspentOutputs))
	m.Handle("/reset", resetAllowed(needConfig(a.reset)))

	m.Handle(networkRPCPrefix+"submit", needConfig(func(ctx context.Context, tx *legacy.Tx) error {
		return a.submitter.Submit(ctx, tx)
	}))
	m.Handle(networkRPCPrefix+"get-block", needConfig(a.getBlockRPC))
	m.Handle(networkRPCPrefix+"get-snapshot-info", needConfig(a.getSnapshotInfoRPC))
	m.Handle(networkRPCPrefix+"get-snapshot", http.HandlerFunc(a.getSnapshotRPC))
	m.Handle(networkRPCPrefix+"signer/sign-block", needConfig(a.leaderSignHandler(a.signer)))
	m.Handle(networkRPCPrefix+"block-height", needConfig(func(ctx context.Context) map[string]uint64 {
		h := a.chain.Height()
		return map[string]uint64{
			"block_height": h,
		}
	}))

	m.Handle("/list-authorization-grant", jsonHandler(a.listGrants))
	m.Handle("/create-authorization-grant", jsonHandler(a.createGrant))
	m.Handle("/delete-authorization-grant", jsonHandler(a.deleteGrant))
	m.Handle("/create-access-token", jsonHandler(a.createAccessToken))
	m.Handle("/list-access-tokens", jsonHandler(a.listAccessTokens))
	m.Handle("/delete-access-token", jsonHandler(a.deleteAccessToken))
	m.Handle("/configure", jsonHandler(a.configure))
	m.Handle("/info", jsonHandler(a.info))

	m.Handle("/debug/vars", expvar.Handler())
	m.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	m.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	m.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	m.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	latencyHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if l := latency(m, req); l != nil {
			defer l.RecordSince(time.Now())
		}
		m.ServeHTTP(w, req)
	})

	handler := a.authzHandler(m, latencyHandler)
	handler = a.authnHandler(handler)
	handler = maxBytes(handler) // TODO(tessr): consider moving this to non-core specific mux
	handler = webAssetsHandler(handler)
	handler = healthHandler(handler)
	for _, l := range a.requestLimits {
		handler = limit.Handler(handler, alwaysError(errRateLimited), l.perSecond, l.burst, l.key)
	}
	handler = gzip.Handler{Handler: handler}
	handler = coreCounter(handler)
	handler = timeoutContextHandler(handler)
	if a.config != nil && a.config.BlockchainId != nil {
		handler = blockchainIDHandler(handler, a.config.BlockchainId.String())
	}
	a.handler = handler
}

// Used as a request object for api queries
type requestQuery struct {
	Filter       string        `json:"filter,omitempty"`
	FilterParams []interface{} `json:"filter_params,omitempty"`
	SumBy        []string      `json:"sum_by,omitempty"`
	PageSize     int           `json:"page_size"`

	// AscLongPoll and Timeout are used by /list-transactions
	// to facilitate notifications.
	AscLongPoll bool          `json:"ascending_with_long_poll,omitempty"`
	Timeout     json.Duration `json:"timeout"`

	// After is a completely opaque cursor, indicating that only
	// items in the result set after the one identified by `After`
	// should be included. It has no relationship to time.
	After string `json:"after"`

	// These two are used for time-range queries like /list-transactions
	StartTimeMS uint64 `json:"start_time,omitempty"`
	EndTimeMS   uint64 `json:"end_time,omitempty"`

	// This is used for point-in-time queries like /list-balances
	// TODO(bobg): Different request structs for endpoints with different needs
	TimestampMS uint64 `json:"timestamp,omitempty"`

	// This is used for filtering results from /list-access-tokens
	// Value must be "client" or "network"
	Type string `json:"type"`

	// Aliases is used to filter results from /mockshm/list-keys
	Aliases []string `json:"aliases,omitempty"`
}

// Used as a response object for api queries
type page struct {
	Items    interface{}  `json:"items"`
	Next     requestQuery `json:"next"`
	LastPage bool         `json:"last_page"`
}

func (a *API) authnHandler(handler http.Handler) http.Handler {
	auth := authn.NewAPI(a.accessTokens, networkRPCPrefix)
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req, err := auth.Authenticate(req)
		if err != nil {
			errorFormatter.Write(req.Context(), rw, errNotAuthenticated)
			return
		}
		handler.ServeHTTP(rw, req)
	})
}

func (a *API) authzHandler(mux *http.ServeMux, handler http.Handler) http.Handler {
	auth := authz.NewAuthorizer(a.raftDB, grantPrefix, policyByRoute)
	auth.GrantInternal(a.internalSubj)
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// return failure early if this path isn't legit
		if _, pat := mux.Handler(req); pat != req.URL.Path {
			errorFormatter.Write(req.Context(), rw, errNotFound)
			return
		}
		err := auth.Authorize(req)
		if errors.Root(err) == authz.ErrNotAuthorized {
			// TODO(kr): remove this workaround once dashboard
			// knows how to handle ErrNotAuthorized (CH011).
			err = errors.Sub(errNotAuthenticated, err)
		}
		if err != nil {
			errorFormatter.Write(req.Context(), rw, err)
			return
		}
		handler.ServeHTTP(rw, req)
	})
}

// timeoutContextHandler propagates the timeout, if any, provided as a header
// in the http request.
func timeoutContextHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		timeout, err := time.ParseDuration(req.Header.Get(rpc.HeaderTimeout))
		if err != nil {
			handler.ServeHTTP(w, req) // unmodified
			return
		}

		ctx := req.Context()
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		handler.ServeHTTP(w, req.WithContext(ctx))
	})
}

// blockchainIDHandler adds the Blockchain-ID HTTP header to all
// requests.
func blockchainIDHandler(handler http.Handler, blockchainID string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set(rpc.HeaderBlockchainID, blockchainID)
		handler.ServeHTTP(w, req)
	})
}

func webAssetsHandler(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", static.Handler{
		Assets:  dashboard.Files,
		Default: "index.html",
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

func (a *API) leaderSignHandler(f func(context.Context, *legacy.Block) ([]byte, error)) func(context.Context, *legacy.Block) ([]byte, error) {
	return func(ctx context.Context, b *legacy.Block) ([]byte, error) {
		if f == nil {
			return nil, errNotFound // TODO(kr): is this really the right error here?
		}
		if a.leader.State() == leader.Leading {
			return f(ctx, b)
		}
		var resp []byte
		err := a.forwardToLeader(ctx, "/rpc/signer/sign-block", b, &resp)
		return resp, err
	}
}

// forwardToLeader forwards the current request to the core's leader
// process. It propagates the same credentials used in the current
// request. For that reason, it cannot be used outside of a request-
// handling context.
func (a *API) forwardToLeader(ctx context.Context, path string, body interface{}, resp interface{}) error {
	addr, err := a.leader.Address(ctx)
	if err != nil {
		return errors.Wrap(err)
	}

	// Don't infinite loop if the leader's address is our own address.
	// This is possible if we just became the leader. The client should
	// just retry.
	if addr == a.addr {
		return errLeaderElection
	}

	l := &rpc.Client{
		BaseURL: "http://" + addr,
		Client:  a.httpClient,
	}
	if a.forwardUsingTLS {
		l.BaseURL = "https://" + addr
	}

	// Forward the request credentials if we have them.
	// TODO(jackson): Don't use the incoming request's credentials and
	// have an alternative authentication scheme between processes of the
	// same Core. For now, we only call the leader for the purpose of
	// forwarding a request, so this is OK.
	req := httpjson.Request(ctx)
	user, pass, ok := req.BasicAuth()
	if ok {
		l.AccessToken = fmt.Sprintf("%s:%s", user, pass)
	}

	return l.Call(ctx, path, body, resp)
}

func healthHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/health" {
			return
		}
		handler.ServeHTTP(w, req)
	})
}

func jsonHandler(f interface{}) http.Handler {
	h, err := httpjson.Handler(f, errorFormatter.Write)
	if err != nil {
		panic(err)
	}
	return h
}

func alwaysError(err error) http.Handler {
	return jsonHandler(func() error { return err })
}

func batchRecover(ctx context.Context, v *interface{}) {
	if r := recover(); r != nil {
		var err error
		if recoveredErr, ok := r.(error); ok {
			err = recoveredErr
		} else {
			err = fmt.Errorf("panic with %T", r)
		}
		err = errors.Wrap(err)
		*v = err
	}

	if *v == nil {
		return
	}
	// Convert errors into error responses (including errors
	// from recovered panics above).
	if err, ok := (*v).(error); ok {
		errorFormatter.Log(ctx, err)
		*v = errorFormatter.Format(err)
	}
}
