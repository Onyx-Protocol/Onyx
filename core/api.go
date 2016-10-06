// Package core provides http handlers for all Chain operations.
package core

import (
	"context"
	"net/http"
	"time"

	"chain/core/leader"
	"chain/core/mockhsm"
	"chain/core/query"
	"chain/encoding/json"
	"chain/errors"
	"chain/net/http/authn"
	"chain/net/http/httpjson"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
)

const (
	defGenericPageSize = 100
)

// TODO(kr): change this to "network" or something.
const networkRPCPrefix = "/rpc/"

var errNotFound = errors.New("not found")

// BlockSignerFunc is the type used for providing a function
// to the core for handling sign block requests.
type BlockSignerFunc func(context.Context, *bc.Block) ([]byte, error)

// Handler returns a handler that serves the Chain HTTP API.
func Handler(
	c *protocol.Chain,
	signer BlockSignerFunc,
	hsm *mockhsm.HSM,
	indexer *query.Indexer,
	config *Config,
	altAuth func(*http.Request) bool,
) http.Handler {
	a := &api{
		c:       c,
		hsm:     hsm,
		indexer: indexer,
		config:  config,
	}
	needConfig := jsonHandler
	if config == nil {
		needConfig = func(f interface{}) http.Handler {
			return alwaysError(errUnconfigured)
		}
	}

	m := http.NewServeMux()
	m.Handle("/", alwaysError(errNotFound))

	m.Handle("/create-account", needConfig(createAccount))
	m.Handle("/create-asset", needConfig(a.createAsset))
	m.Handle("/build-transaction", needConfig(build))
	m.Handle("/submit-transaction", needConfig(a.submit))
	m.Handle("/create-control-program", needConfig(createControlProgram))
	m.Handle("/create-transaction-consumer", needConfig(a.createTxConsumer))
	m.Handle("/get-transaction-consumer", needConfig(getTxConsumer))
	m.Handle("/update-transaction-consumer", needConfig(updateTxConsumer))
	m.Handle("/delete-transaction-consumer", needConfig(deleteTxConsumer))
	m.Handle("/mockhsm/create-key", needConfig(a.mockhsmCreateKey))
	m.Handle("/mockhsm/list-keys", needConfig(a.mockhsmListKeys))
	m.Handle("/mockhsm/delkey", needConfig(a.mockhsmDelKey))
	m.Handle("/mockhsm/sign-transaction", needConfig(a.mockhsmSignTemplates))
	m.Handle("/list-accounts", needConfig(a.listAccounts))
	m.Handle("/list-assets", needConfig(a.listAssets))
	m.Handle("/list-transaction-consumers", needConfig(a.listTxConsumers))
	m.Handle("/list-transactions", needConfig(a.listTransactions))
	m.Handle("/list-balances", needConfig(a.listBalances))
	m.Handle("/list-unspent-outputs", needConfig(a.listUnspentOutputs))
	m.Handle("/reset", needConfig(a.reset))

	// V3 DEPRECATED
	m.Handle("/v3/transact/cancel-reservation", needConfig(cancelReservation))

	m.Handle(networkRPCPrefix+"submit", needConfig(a.c.AddTx))
	m.Handle(networkRPCPrefix+"get-blocks", needConfig(a.getBlocksRPC))
	m.Handle(networkRPCPrefix+"signer/sign-block", needConfig(leaderSignHandler(signer)))
	m.Handle(networkRPCPrefix+"block-height", needConfig(func(ctx context.Context) map[string]uint64 {
		h := a.c.Height()
		return map[string]uint64{
			"block_height": h,
		}
	}))

	m.Handle("/create-access-token", jsonHandler(createAccessToken))
	m.Handle("/list-access-tokens", jsonHandler(listAccessTokens))
	m.Handle("/delete-access-token", jsonHandler(deleteAccessToken))
	m.Handle("/configure", jsonHandler(a.configure))
	m.Handle("/info", jsonHandler(a.info))

	latencyHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if l := latency(m, req); l != nil {
			defer l.RecordSince(time.Now())
		}
		m.ServeHTTP(w, req)
	})

	return authn.BasicHandler{
		Auth: (&apiAuthn{
			tokenMap: make(map[string]tokenResult),
			alt:      altAuth,
		}).auth,
		Next:  latencyHandler,
		Realm: "Chain Core API",
	}
}

// Config encapsulates Core-level, persistent configuration options.
type Config struct {
	IsSigner             bool    `json:"is_signer"`
	IsGenerator          bool    `json:"is_generator"`
	BlockchainID         bc.Hash `json:"blockchain_id"`
	GeneratorURL         string  `json:"generator_url"`
	GeneratorAccessToken string  `json:"generator_access_token"`
	ConfiguredAt         time.Time
	BlockXPub            string         `json:"block_xpub"`
	Signers              []ConfigSigner `json:"block_signer_urls"`
	Quorum               int
	MaxIssuanceWindow    time.Duration
}

type ConfigSigner struct {
	Pubkey json.HexBytes `json:"pubkey"`
	URL    string        `json:"url"`
}

type api struct {
	c       *protocol.Chain
	hsm     *mockhsm.HSM
	indexer *query.Indexer
	config  *Config
}

// Used as a request object for api queries
type requestQuery struct {
	Filter       string        `json:"filter,omitempty"`
	FilterParams []interface{} `json:"filter_params,omitempty"`
	SumBy        []string      `json:"sum_by,omitempty"`
	PageSize     int           `json:"page_size"`

	// AscLongPoll and Timeout are used by /list-transactions
	// to facilitate notifications.
	AscLongPoll bool              `json:"ascending_with_long_poll,omitempty"`
	Timeout     httpjson.Duration `json:"timeout"`

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
}

// Used as a response object for api queries
type page struct {
	Items    interface{}  `json:"items"`
	Next     requestQuery `json:"next"`
	LastPage bool         `json:"last_page"`
}

func leaderSignHandler(f BlockSignerFunc) BlockSignerFunc {
	return func(ctx context.Context, b *bc.Block) ([]byte, error) {
		if f == nil {
			return nil, errNotFound // TODO(kr): is this really the right error here?
		}
		if leader.IsLeading() {
			return f(ctx, b)
		}
		var resp []byte
		err := callLeader(ctx, "/rpc/signer/sign-block", b, &resp)
		return resp, err
	}
}

func callLeader(ctx context.Context, path string, body interface{}, resp interface{}) error {
	addr, err := leader.Address(ctx)
	if err != nil {
		return errors.Wrap(err)
	}

	l := &rpc.Client{
		BaseURL: "https://" + addr,
		// TODO(tessr): Auth.
	}

	return l.Call(ctx, path, body, &resp)
}
