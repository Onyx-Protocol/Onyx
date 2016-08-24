// Package core provides http handlers for all Chain operations.
package core

import (
	"net/http"

	"chain/core/blocksigner"
	"chain/core/generator"
	"chain/core/mockhsm"
	"chain/core/query"
	"chain/core/txdb"
	"chain/protocol"
)

const (
	defGenericPageSize = 100
)

// Handler returns a handler that serves the Chain HTTP API.
func Handler(
	apiSecret string,
	fc *protocol.FC,
	generatorConfig *generator.Config,
	signer *blocksigner.Signer,
	store *txdb.Store,
	pool *txdb.Pool,
	hsm *mockhsm.HSM,
	indexer *query.Indexer,
) http.Handler {
	a := &api{
		fc:        fc,
		store:     store,
		pool:      pool,
		generator: generatorConfig,
		hsm:       hsm,
		indexer:   indexer,
	}

	m := http.NewServeMux()
	m.Handle("/", apiAuthn(apiSecret, a.handler()))
	m.Handle("/rpc/", rpcAuthn(rpcAuthedHandler(generatorConfig, signer)))
	return m
}

type api struct {
	fc        *protocol.FC
	store     *txdb.Store
	pool      *txdb.Pool
	generator *generator.Config
	hsm       *mockhsm.HSM
	indexer   *query.Indexer
}

// Used as a request object for api queries
type requestQuery struct {
	Cursor string `json:"cursor"`

	// These two are used for time-range queries like /list-transactions
	StartTimeMS uint64 `json:"start_time,omitempty"`
	EndTimeMS   uint64 `json:"end_time,omitempty"`

	// This is used for point-in-time queries like /list-balances
	// TODO(bobg): Different request structs for endpoints with different needs
	TimestampMS uint64 `json:"timestamp,omitempty"`

	IndexID      string        `json:"index_id,omitempty"`
	IndexAlias   string        `json:"index_alias,omitempty"`
	Filter       string        `json:"filter,omitempty"`
	FilterParams []interface{} `json:"filter_params,omitempty"`
	SumBy        []string      `json:"sum_by,omitempty"`
}

// Used as a response object for api queries
type page struct {
	Items    interface{}  `json:"items"`
	LastPage bool         `json:"last_page"`
	Query    requestQuery `json:"query"`
}

func (a *api) handler() http.Handler {
	m := http.NewServeMux()

	// Accounts
	m.Handle("/create-account", jsonHandler(createAccount))
	m.Handle("/set-account-tags", jsonHandler(setAccountTags))
	m.Handle("/archive-account", jsonHandler(archiveAccount))

	// Assets
	m.Handle("/create-asset", jsonHandler(a.createAsset))
	m.Handle("/set-asset-tags", jsonHandler(setAssetTags))
	m.Handle("/archive-asset", jsonHandler(archiveAsset))

	// Transactions
	m.Handle("/build-transaction-template", jsonHandler(build))
	m.Handle("/submit-transaction-template", jsonHandler(a.submit))
	m.Handle("/create-control-program", jsonHandler(createControlProgram))

	// MockHSM endpoints
	m.Handle("/mockhsm/create-key", jsonHandler(a.mockhsmCreateKey))
	m.Handle("/mockhsm/list-keys", jsonHandler(a.mockhsmListKeys))
	m.Handle("/mockhsm/delkey", jsonHandler(a.mockhsmDelKey))
	m.Handle("/mockhsm/sign-transaction-template", jsonHandler(a.mockhsmSignTemplates))

	// Transaction indexes & querying
	m.Handle("/create-index", jsonHandler(a.createIndex))
	m.Handle("/list-indexes", jsonHandler(a.listIndexes))
	m.Handle("/list-accounts", jsonHandler(a.listAccounts))
	m.Handle("/list-assets", jsonHandler(a.listAssets))
	m.Handle("/list-transactions", jsonHandler(a.listTransactions))
	m.Handle("/list-balances", jsonHandler(a.listBalances))
	m.Handle("/list-unspent-outputs", jsonHandler(a.listUnspentOutputs))

	// V3 DEPRECATED
	m.Handle("/v3/transact/cancel-reservation", jsonHandler(cancelReservation))

	return m
}

func rpcAuthedHandler(generator *generator.Config, signer *blocksigner.Signer) http.Handler {
	m := http.NewServeMux()

	if generator != nil {
		m.Handle("/rpc/generator/submit", jsonHandler(generator.Submit))
		m.Handle("/rpc/generator/get-blocks", jsonHandler(generator.GetBlocks))
	}
	if signer != nil {
		m.Handle("/rpc/signer/sign-block", jsonHandler(signer.SignBlock))
	}

	return m
}
