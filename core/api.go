// Package core implements Chain Core and its API.
package core

import (
	"context"
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
	"chain/core/leader"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/rpc"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/core/txfeed"
	"chain/database/pg"
	"chain/encoding/json"
	"chain/errors"
	"chain/generated/dashboard"
	"chain/generated/docs"
	"chain/net/http/gzip"
	"chain/net/http/httpjson"
	"chain/net/http/limit"
	"chain/net/http/reqid"
	"chain/net/http/static"
	"chain/protocol"
	"chain/protocol/bc"
)

const (
	defGenericPageSize = 100
)

// TODO(kr): change this to "network" or something.
const networkRPCPrefix = "/rpc/"

var (
	errNotFound       = errors.New("not found")
	errRateLimited    = errors.New("request limit exceeded")
	errLeaderElection = errors.New("no leader; pending election")
)

// Handler serves the Chain HTTP API
type API struct {
	Chain         *protocol.Chain
	Store         *txdb.Store
	PinStore      *pin.Store
	Assets        *asset.Registry
	Accounts      *account.Manager
	Indexer       *query.Indexer
	TxFeeds       *txfeed.Tracker
	AccessTokens  *accesstoken.CredentialStore
	Config        *config.Config
	Submitter     txbuilder.Submitter
	DB            pg.DB
	Addr          string
	AltAuth       func(*http.Request) bool
	Signer        func(context.Context, *bc.Block) ([]byte, error)
	RequestLimits []RequestLimit

	healthMu     sync.Mutex
	healthErrors map[string]interface{}
}

type RequestLimit struct {
	Key       func(*http.Request) string
	Burst     int
	PerSecond int
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
	if a.Config == nil {
		return func(f interface{}) http.Handler {
			return alwaysError(errUnconfigured)
		}
	}
	return jsonHandler
}

func Handler(a *API, register func(*http.ServeMux, *API)) http.Handler {
	// Setup the muxer.
	needConfig := a.needConfig()

	devOnly := func(h http.Handler) http.Handler { return h }
	if config.Production {
		devOnly = func(h http.Handler) http.Handler { return alwaysError(errProduction) }
	}

	m := http.NewServeMux()
	m.Handle("/", alwaysError(errNotFound))

	m.Handle("/create-account", needConfig(a.createAccount))
	m.Handle("/create-asset", needConfig(a.createAsset))
	m.Handle("/build-transaction", needConfig(a.build))
	m.Handle("/submit-transaction", needConfig(a.submit))
	m.Handle("/create-control-program", needConfig(a.createControlProgram)) // DEPRECATED
	m.Handle("/create-account-receiver", needConfig(a.createAccountReceiver))
	m.Handle("/create-transaction-feed", needConfig(a.createTxFeed))
	m.Handle("/get-transaction-feed", needConfig(a.getTxFeed))
	m.Handle("/update-transaction-feed", needConfig(a.updateTxFeed))
	m.Handle("/delete-transaction-feed", needConfig(a.deleteTxFeed))
	m.Handle("/mockhsm", alwaysError(errProduction))
	m.Handle("/list-accounts", needConfig(a.listAccounts))
	m.Handle("/list-assets", needConfig(a.listAssets))
	m.Handle("/list-transaction-feeds", needConfig(a.listTxFeeds))
	m.Handle("/list-transactions", needConfig(a.listTransactions))
	m.Handle("/list-balances", needConfig(a.listBalances))
	m.Handle("/list-unspent-outputs", needConfig(a.listUnspentOutputs))
	m.Handle("/reset", devOnly(needConfig(a.reset)))

	m.Handle(networkRPCPrefix+"submit", needConfig(func(ctx context.Context, tx *bc.Tx) error {
		return a.Submitter.Submit(ctx, tx)
	}))
	m.Handle(networkRPCPrefix+"get-blocks", needConfig(a.getBlocksRPC)) // DEPRECATED: use get-block instead
	m.Handle(networkRPCPrefix+"get-block", needConfig(a.getBlockRPC))
	m.Handle(networkRPCPrefix+"get-snapshot-info", needConfig(a.getSnapshotInfoRPC))
	m.Handle(networkRPCPrefix+"get-snapshot", http.HandlerFunc(a.getSnapshotRPC))
	m.Handle(networkRPCPrefix+"signer/sign-block", needConfig(a.leaderSignHandler(a.Signer)))
	m.Handle(networkRPCPrefix+"block-height", needConfig(func(ctx context.Context) map[string]uint64 {
		h := a.Chain.Height()
		return map[string]uint64{
			"block_height": h,
		}
	}))

	m.Handle("/create-access-token", jsonHandler(a.createAccessToken))
	m.Handle("/list-access-tokens", jsonHandler(a.listAccessTokens))
	m.Handle("/delete-access-token", jsonHandler(a.deleteAccessToken))
	m.Handle("/configure", jsonHandler(a.configure))
	m.Handle("/info", jsonHandler(a.info))

	m.Handle("/debug/vars", http.HandlerFunc(expvarHandler))
	m.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	m.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	m.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	m.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	if register != nil {
		register(m, a)
	}

	latencyHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if l := latency(m, req); l != nil {
			defer l.RecordSince(time.Now())
		}
		m.ServeHTTP(w, req)
	})

	var handler = (&apiAuthn{
		tokens:   a.AccessTokens,
		tokenMap: make(map[string]tokenResult),
		alt:      a.AltAuth,
	}).handler(latencyHandler)
	handler = maxBytes(handler)
	handler = webAssetsHandler(handler)
	handler = healthHandler(handler)
	for _, l := range a.RequestLimits {
		handler = limit.Handler(handler, alwaysError(errRateLimited), l.PerSecond, l.Burst, l.Key)
	}
	handler = gzip.Handler{Handler: handler}
	handler = coreCounter(handler)
	handler = reqid.Handler(handler)
	handler = timeoutContextHandler(handler)

	return handler
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

func webAssetsHandler(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", static.Handler{
		Assets:  dashboard.Files,
		Default: "index.html",
	}))
	mux.Handle("/docs/", http.StripPrefix("/docs/", static.Handler{
		Assets: docs.Files,
		Index:  "index.html",
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

func (a *API) leaderSignHandler(f func(context.Context, *bc.Block) ([]byte, error)) func(context.Context, *bc.Block) ([]byte, error) {
	return func(ctx context.Context, b *bc.Block) ([]byte, error) {
		if f == nil {
			return nil, errNotFound // TODO(kr): is this really the right error here?
		}
		if leader.IsLeading() {
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
	addr, err := leader.Address(ctx, a.DB)
	if err != nil {
		return errors.Wrap(err)
	}

	// Don't infinite loop if the leader's address is our own address.
	// This is possible if we just became the leader. The client should
	// just retry.
	if addr == a.Addr {
		return errLeaderElection
	}

	leaderURL := "http://" + addr
	if config.TLS {
		// Issue #674: If cored is configured to use TLS, assume the
		// leader is reachable via https. This is a fix for the 1.1.1 release.
		leaderURL = "https://" + addr
	}
	l := &rpc.Client{BaseURL: leaderURL}

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

// expvarHandler is copied from the expvar package.
// TODO(jackson): In Go 1.8, use expvar.Handler.
// https://go-review.googlesource.com/#/c/24722/
func expvarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

func healthHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/health" {
			return
		}
		handler.ServeHTTP(w, req)
	})
}
