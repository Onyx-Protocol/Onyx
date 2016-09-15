// Package core provides http handlers for all Chain operations.
package core

import (
	"context"
	"net/http"
	"time"

	"chain/core/generator"
	"chain/core/mockhsm"
	"chain/core/query"
	"chain/protocol"
	"chain/protocol/bc"
)

const (
	defGenericPageSize = 100
)

// Handler returns a handler that serves the Chain HTTP API.
func Handler(
	apiSecret, rpcSecret string,
	c *protocol.Chain,
	signer func(context.Context, *bc.Block) ([]byte, error),
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
		m.Handle("/", apiAuthn(apiSecret, a.handler()))
		m.Handle("/rpc/", apiAuthn(rpcSecret, rpcAuthedHandler(c, signer)))
		m.Handle("/configure", apiAuthn(apiSecret, alwaysError(errAlreadyConfigured)))
	} else {
		m.Handle("/", apiAuthn(apiSecret, alwaysError(errUnconfigured)))
		m.Handle("/configure", apiAuthn(apiSecret, jsonHandler(configure)))
	}
	m.Handle("/info", jsonHandler(a.info))
	return m
}

// Config encapsulates Core-level, persistent configuration options.
type Config struct {
	IsSigner         bool    `json:"is_signer"`
	IsGenerator      bool    `json:"is_generator"`
	InitialBlockHash bc.Hash `json:"initial_block_hash"`
	GeneratorURL     string  `json:"generator_url"`
	ConfiguredAt     time.Time
	BlockXPub        string `json:"block_xpub"`
}

type api struct {
	c       *protocol.Chain
	hsm     *mockhsm.HSM
	indexer *query.Indexer
	config  *Config
}

// Used as a request object for api queries
type requestQuery struct {
	After string `json:"after"`

	// These two are used for time-range queries like /list-transactions
	StartTimeMS uint64 `json:"start_time,omitempty"`
	EndTimeMS   uint64 `json:"end_time,omitempty"`

	// This is used for point-in-time queries like /list-balances
	// TODO(bobg): Different request structs for endpoints with different needs
	TimestampMS uint64 `json:"timestamp,omitempty"`

	// This is used by /list-transactions.
	Order string `json:"order,omitempty"`

	Filter       string        `json:"filter,omitempty"`
	FilterParams []interface{} `json:"filter_params,omitempty"`
	SumBy        []string      `json:"sum_by,omitempty"`
}

// Used as a response object for api queries
type page struct {
	Items    interface{}  `json:"items"`
	LastPage bool         `json:"last_page"`
	Next     requestQuery `json:"next"`
}

func (a *api) handler() http.Handler {
	m := http.NewServeMux()

	// Accounts
	m.Handle("/create-account", jsonHandler(createAccount))
	m.Handle("/archive-account", jsonHandler(archiveAccount))

	// Assets
	m.Handle("/create-asset", jsonHandler(a.createAsset))
	m.Handle("/archive-asset", jsonHandler(archiveAsset))

	// Transactions
	m.Handle("/build-transaction", jsonHandler(build))
	m.Handle("/submit-transaction", jsonHandler(a.submit))
	m.Handle("/create-control-program", jsonHandler(createControlProgram))

	// Cursors
	m.Handle("/create-cursor", jsonHandler(createCursor))

	// MockHSM endpoints
	m.Handle("/mockhsm/create-key", jsonHandler(a.mockhsmCreateKey))
	m.Handle("/mockhsm/list-keys", jsonHandler(a.mockhsmListKeys))
	m.Handle("/mockhsm/delkey", jsonHandler(a.mockhsmDelKey))
	m.Handle("/mockhsm/sign-transaction", jsonHandler(a.mockhsmSignTemplates))

	// Transaction querying
	m.Handle("/list-accounts", jsonHandler(a.listAccounts))
	m.Handle("/list-assets", jsonHandler(a.listAssets))
	m.Handle("/list-transactions", jsonHandler(a.listTransactions))
	m.Handle("/list-balances", jsonHandler(a.listBalances))
	m.Handle("/list-unspent-outputs", jsonHandler(a.listUnspentOutputs))

	m.Handle("/reset", jsonHandler(a.reset))

	// V3 DEPRECATED
	m.Handle("/v3/transact/cancel-reservation", jsonHandler(cancelReservation))

	return m
}

func rpcAuthedHandler(c *protocol.Chain, signer func(context.Context, *bc.Block) ([]byte, error)) http.Handler {
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
		m.Handle("/rpc/signer/sign-block", jsonHandler(signer))
	}

	return m
}
