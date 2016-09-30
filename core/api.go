// Package core provides http handlers for all Chain operations.
package core

import (
	"context"
	"net/http"
	"sync"
	"time"

	"chain/core/generator"
	"chain/core/leader"
	"chain/core/mockhsm"
	"chain/core/query"
	"chain/errors"
	"chain/net/http/httpjson"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
)

const (
	defGenericPageSize = 100
)

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
) http.Handler {
	a := &api{
		c:       c,
		hsm:     hsm,
		indexer: indexer,
		config:  config,
	}

	m := http.NewServeMux()
	if config != nil {
		authn := &apiAuthn{config: config, tokenMap: make(map[string]tokenResult)}
		m.Handle("/", authn.Handler("client", a.handler()))
		m.Handle("/rpc/", authn.Handler("network", rpcAuthedHandler(c, signer)))
		m.Handle("/configure", authn.Handler("client", alwaysError(errAlreadyConfigured)))
		m.Handle("/info", authn.Handler("client", jsonHandler(a.info)))
	} else {
		m.Handle("/", alwaysError(errUnconfigured))
		m.Handle("/configure", jsonHandler(configure))
		m.Handle("/info", jsonHandler(a.info))
		registerAccessTokens(m)
	}
	return m
}

// Config encapsulates Core-level, persistent configuration options.
type Config struct {
	IsSigner     bool    `json:"is_signer"`
	IsGenerator  bool    `json:"is_generator"`
	BlockchainID bc.Hash `json:"blockchain_id"`
	GeneratorURL string  `json:"generator_url"`
	ConfiguredAt time.Time
	BlockXPub    string `json:"block_xpub"`

	authedMu      sync.Mutex // protects the following
	ClientAuthed  bool       `json:"require_client_access_tokens"`
	NetworkAuthed bool       `json:"require_network_access_tokens"`
}

func (c *Config) authEnabled(typ string) bool {
	return (typ == "client" && c.isClientAuthed()) || (typ == "network" && c.isNetworkAuthed())
}

func (c *Config) isClientAuthed() bool {
	c.authedMu.Lock()
	defer c.authedMu.Unlock()
	return c.ClientAuthed
}

func (c *Config) isNetworkAuthed() bool {
	c.authedMu.Lock()
	defer c.authedMu.Unlock()
	return c.NetworkAuthed
}

func (c *Config) setClientAuthed(a bool) {
	c.authedMu.Lock()
	defer c.authedMu.Unlock()
	c.ClientAuthed = a
}

func (c *Config) setNetworkAuthed(a bool) {
	c.authedMu.Lock()
	defer c.authedMu.Unlock()
	c.NetworkAuthed = a
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

	// Order and Timeout are used by /list-transactions
	// to facilitate notifications.
	Order   string            `json:"order,omitempty"`
	Timeout httpjson.Duration `json:"timeout"`

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

func (a *api) handler() http.Handler {
	m := http.NewServeMux()

	// Accounts
	m.Handle("/create-account", jsonHandler(createAccount))

	// Assets
	m.Handle("/create-asset", jsonHandler(a.createAsset))

	// Transactions
	m.Handle("/build-transaction", jsonHandler(build))
	m.Handle("/submit-transaction", jsonHandler(a.submit))
	m.Handle("/create-control-program", jsonHandler(createControlProgram))

	// Cursors
	m.Handle("/create-cursor", jsonHandler(a.createCursor))
	m.Handle("/get-cursor", jsonHandler(getCursor))
	m.Handle("/update-cursor", jsonHandler(updateCursor))
	m.Handle("/delete-cursor", jsonHandler(deleteCursor))

	// MockHSM endpoints
	m.Handle("/mockhsm/create-key", jsonHandler(a.mockhsmCreateKey))
	m.Handle("/mockhsm/list-keys", jsonHandler(a.mockhsmListKeys))
	m.Handle("/mockhsm/delkey", jsonHandler(a.mockhsmDelKey))
	m.Handle("/mockhsm/sign-transaction", jsonHandler(a.mockhsmSignTemplates))

	// Transaction querying
	m.Handle("/list-accounts", jsonHandler(a.listAccounts))
	m.Handle("/list-assets", jsonHandler(a.listAssets))
	m.Handle("/list-cursors", jsonHandler(a.listCursors))
	m.Handle("/list-transactions", jsonHandler(a.listTransactions))
	m.Handle("/list-balances", jsonHandler(a.listBalances))
	m.Handle("/list-unspent-outputs", jsonHandler(a.listUnspentOutputs))

	m.Handle("/update-configuration", jsonHandler(a.updateConfig))
	m.Handle("/reset", jsonHandler(a.reset))

	// V3 DEPRECATED
	m.Handle("/v3/transact/cancel-reservation", jsonHandler(cancelReservation))

	registerAccessTokens(m)

	m.Handle("/", alwaysError(errNotFound))

	return m
}

func registerAccessTokens(m *http.ServeMux) {
	m.Handle("/create-access-token", jsonHandler(createAccessToken))
	m.Handle("/list-access-tokens", jsonHandler(listAccessTokens))
	m.Handle("/delete-access-token", jsonHandler(deleteAccessToken))
}

func rpcAuthedHandler(c *protocol.Chain, signer BlockSignerFunc) http.Handler {
	m := http.NewServeMux()

	m.Handle("/rpc/submit", jsonHandler(c.AddTx))
	m.Handle("/rpc/get-blocks", jsonHandler(func(ctx context.Context, h uint64) ([]*bc.Block, error) {
		return generator.GetBlocks(ctx, c, h)
	}))
	m.Handle("/rpc/block-height", jsonHandler(func(ctx context.Context) map[string]uint64 {
		h := c.Height()
		return map[string]uint64{
			"block_height": h,
		}
	}))

	if signer != nil {
		m.Handle("/rpc/signer/sign-block", jsonHandler(leaderSignHandler(signer)))
	}

	return m
}

func leaderSignHandler(f BlockSignerFunc) BlockSignerFunc {
	return func(ctx context.Context, b *bc.Block) ([]byte, error) {
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
