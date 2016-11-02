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
	"chain/core/mockhsm"
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
type Handler struct {
	Chain         *protocol.Chain
	Store         *txdb.Store
	PinStore      *pin.Store
	Assets        *asset.Registry
	Accounts      *account.Manager
	HSM           *mockhsm.HSM
	Indexer       *query.Indexer
	TxFeeds       *txfeed.Tracker
	AccessTokens  *accesstoken.CredentialStore
	Config        *config.Config
	DB            pg.DB
	Addr          string
	AltAuth       func(*http.Request) bool
	Signer        func(context.Context, *bc.Block) ([]byte, error)
	RequestLimits []RequestLimit

	once           sync.Once
	handler        http.Handler
	actionDecoders map[string]func(data []byte) (txbuilder.Action, error)

	healthMu     sync.Mutex
	healthErrors map[string]interface{}
}

type RequestLimit struct {
	Key       func(*http.Request) string
	Burst     int
	PerSecond int
}

func maxBytes(h http.Handler) http.Handler {
	const maxReqSize = 1e6 // 1MB
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// A block can easily be bigger than maxReqSize, but everything
		// else should be pretty small.
		if req.URL.Path != networkRPCPrefix+"signer/sign-block" {
			req.Body = http.MaxBytesReader(w, req.Body, maxReqSize)
		}
		h.ServeHTTP(w, req)
	})
}

func (h *Handler) init() {
	// Setup the available transact actions.
	h.actionDecoders = map[string]func(data []byte) (txbuilder.Action, error){
		"control_account":                h.Accounts.DecodeControlAction,
		"control_program":                txbuilder.DecodeControlProgramAction,
		"issue":                          h.Assets.DecodeIssueAction,
		"spend_account":                  h.Accounts.DecodeSpendAction,
		"spend_account_unspent_output":   h.Accounts.DecodeSpendUTXOAction,
		"set_transaction_reference_data": txbuilder.DecodeSetTxRefDataAction,
	}

	// Setup the muxer.
	needConfig := jsonHandler
	if h.Config == nil {
		needConfig = func(f interface{}) http.Handler {
			return alwaysError(errUnconfigured)
		}
	}

	m := http.NewServeMux()
	m.Handle("/", alwaysError(errNotFound))

	m.Handle("/create-account", needConfig(h.createAccount))
	m.Handle("/create-asset", needConfig(h.createAsset))
	m.Handle("/build-transaction", needConfig(h.build))
	m.Handle("/submit-transaction", needConfig(h.submit))
	m.Handle("/create-control-program", needConfig(h.createControlProgram))
	m.Handle("/create-transaction-feed", needConfig(h.createTxFeed))
	m.Handle("/get-transaction-feed", needConfig(h.getTxFeed))
	m.Handle("/update-transaction-feed", needConfig(h.updateTxFeed))
	m.Handle("/delete-transaction-feed", needConfig(h.deleteTxFeed))
	m.Handle("/mockhsm/create-key", needConfig(h.mockhsmCreateKey))
	m.Handle("/mockhsm/list-keys", needConfig(h.mockhsmListKeys))
	m.Handle("/mockhsm/delkey", needConfig(h.mockhsmDelKey))
	m.Handle("/mockhsm/sign-transaction", needConfig(h.mockhsmSignTemplates))
	m.Handle("/list-accounts", needConfig(h.listAccounts))
	m.Handle("/list-assets", needConfig(h.listAssets))
	m.Handle("/list-transaction-feeds", needConfig(h.listTxFeeds))
	m.Handle("/list-transactions", needConfig(h.listTransactions))
	m.Handle("/list-balances", needConfig(h.listBalances))
	m.Handle("/list-unspent-outputs", needConfig(h.listUnspentOutputs))
	m.Handle("/reset", needConfig(h.reset))

	m.Handle(networkRPCPrefix+"submit", needConfig(h.Chain.AddTx))
	m.Handle(networkRPCPrefix+"get-blocks", needConfig(h.getBlocksRPC)) // DEPRECATED: use get-block instead
	m.Handle(networkRPCPrefix+"get-block", needConfig(h.getBlockRPC))
	m.Handle(networkRPCPrefix+"get-snapshot-info", needConfig(h.getSnapshotInfoRPC))
	m.Handle(networkRPCPrefix+"get-snapshot", http.HandlerFunc(h.getSnapshotRPC))
	m.Handle(networkRPCPrefix+"signer/sign-block", needConfig(h.leaderSignHandler(h.Signer)))
	m.Handle(networkRPCPrefix+"block-height", needConfig(func(ctx context.Context) map[string]uint64 {
		h := h.Chain.Height()
		return map[string]uint64{
			"block_height": h,
		}
	}))

	m.Handle("/create-access-token", jsonHandler(h.createAccessToken))
	m.Handle("/list-access-tokens", jsonHandler(h.listAccessTokens))
	m.Handle("/delete-access-token", jsonHandler(h.deleteAccessToken))
	m.Handle("/configure", jsonHandler(h.configure))
	m.Handle("/info", jsonHandler(h.info))

	m.Handle("/debug/vars", http.HandlerFunc(expvarHandler))
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

	var handler = (&apiAuthn{
		tokens:   h.AccessTokens,
		tokenMap: make(map[string]tokenResult),
		alt:      h.AltAuth,
	}).handler(latencyHandler)
	handler = maxBytes(handler)
	handler = webAssetsHandler(handler)
	handler = healthHandler(handler)
	for _, l := range h.RequestLimits {
		handler = limit.Handler(handler, alwaysError(errRateLimited), l.PerSecond, l.Burst, l.Key)
	}
	handler = gzip.Handler{Handler: handler}
	handler = coreCounter(handler)
	handler = reqid.Handler(handler)
	handler = timeoutContextHandler(handler)
	h.handler = handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(h.init)

	h.handler.ServeHTTP(w, r)
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

func (h *Handler) leaderSignHandler(f func(context.Context, *bc.Block) ([]byte, error)) func(context.Context, *bc.Block) ([]byte, error) {
	return func(ctx context.Context, b *bc.Block) ([]byte, error) {
		if f == nil {
			return nil, errNotFound // TODO(kr): is this really the right error here?
		}
		if leader.IsLeading() {
			return f(ctx, b)
		}
		var resp []byte
		err := h.forwardToLeader(ctx, "/rpc/signer/sign-block", b, &resp)
		return resp, err
	}
}

// forwardToLeader forwards the current request to the core's leader
// process. It propagates the same credentials used in the current
// request. For that reason, it cannot be used outside of a request-
// handling context.
func (h *Handler) forwardToLeader(ctx context.Context, path string, body interface{}, resp interface{}) error {
	addr, err := leader.Address(ctx, h.DB)
	if err != nil {
		return errors.Wrap(err)
	}

	// Don't infinite loop if the leader's address is our own address.
	// This is possible if we just became the leader. The client should
	// just retry.
	if addr == h.Addr {
		return errLeaderElection
	}

	// TODO(jackson): If using TLS, use https:// here.
	l := &rpc.Client{
		BaseURL: "http://" + addr,
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

	return l.Call(ctx, path, body, &resp)
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
