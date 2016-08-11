// Package core provides http handlers for all Chain operations.
package core

import (
	"chain/core/blocksigner"
	"chain/core/generator"
	"chain/core/mockhsm"
	"chain/core/query"
	"chain/core/txdb"
	"chain/cos"
	chainhttp "chain/net/http"
	"chain/net/http/httpjson"
	"chain/net/http/pat"
)

const (
	defAccountPageSize = 100
	defGenericPageSize = 100
)

// Handler returns a handler that serves the Chain HTTP API.
func Handler(
	apiSecret string,
	fc *cos.FC,
	generatorConfig *generator.Config,
	signer *blocksigner.Signer,
	store *txdb.Store,
	pool *txdb.Pool,
	hsm *mockhsm.HSM,
	indexer *query.Indexer,
) chainhttp.Handler {
	h := pat.New()
	a := &api{
		fc:        fc,
		store:     store,
		pool:      pool,
		generator: generatorConfig,
		hsm:       hsm,
		indexer:   indexer,
	}

	apiHandler := apiAuthn(apiSecret, a.handler())
	h.Add("GET", "/", apiHandler)
	h.Add("PUT", "/", apiHandler)
	h.Add("POST", "/", apiHandler)
	h.Add("DELETE", "/", apiHandler)

	rpcHandler := chainhttp.HandlerFunc(rpcAuthn(rpcAuthedHandler(generatorConfig, signer)))
	h.Add("GET", "/rpc/", rpcHandler)
	h.Add("PUT", "/rpc/", rpcHandler)
	h.Add("POST", "/rpc/", rpcHandler)
	h.Add("DELETE", "/rpc/", rpcHandler)

	return h
}

type api struct {
	fc        *cos.FC
	store     *txdb.Store
	pool      *txdb.Pool
	generator *generator.Config
	hsm       *mockhsm.HSM
	indexer   *query.Indexer
}

// Used as a request object for api queries
type requestQuery struct {
	Cursor    string        `json:"cursor"`
	StartTime uint64        `json:"start_time,omitempty"`
	EndTime   uint64        `json:"end_time,omitempty"`
	Query     string        `json:"query,omitempty"`
	Index     string        `json:"index,omitempty"`
	Params    []interface{} `json:"params,omitempty"`
}

// Used as a response object for api queries
type page struct {
	Items    interface{}  `json:"items"`
	LastPage bool         `json:"last_page"`
	Query    requestQuery `json:"query"`
}

func (a *api) handler() chainhttp.HandlerFunc {
	h := httpjson.NewServeMux(writeHTTPError)

	// Accounts
	h.HandleFunc("POST", "/list-accounts", listAccounts)
	h.HandleFunc("POST", "/create-account", createAccount)
	h.HandleFunc("POST", "/get-account", getAccount)
	h.HandleFunc("POST", "/set-account-tags", setAccountTags)

	// Assets
	h.HandleFunc("POST", "/list-assets", a.listAssets)
	h.HandleFunc("POST", "/create-asset", a.createAsset)
	h.HandleFunc("POST", "/set-asset-tags", setAssetTags)

	// Transactions
	h.HandleFunc("POST", "/build-transaction-template", build)
	h.HandleFunc("POST", "/submit-transaction-template", a.submit)
	h.HandleFunc("POST", "/create-control-program", createControlProgram)

	// MockHSM endpoints
	h.HandleFunc("POST", "/mockhsm/create-key", a.mockhsmCreateKey)
	h.HandleFunc("POST", "/mockhsm/list-keys", a.mockhsmListKeys)
	h.HandleFunc("POST", "/mockhsm/delkey", a.mockhsmDelKey)
	h.HandleFunc("POST", "/mockhsm/sign-transaction-template", a.mockhsmSignTemplates)

	// Transaction indexes & querying
	h.HandleFunc("POST", "/create-index", a.createIndex)
	h.HandleFunc("POST", "/list-indexes", a.listIndexes)
	h.HandleFunc("POST", "/list-transactions", a.listTransactions)

	// V3 DEPRECATED
	h.HandleFunc("DELETE", "/v3/accounts/:accountID", archiveAccount)
	h.HandleFunc("DELETE", "/v3/assets/:assetID", archiveAsset)
	h.HandleFunc("POST", "/v3/transact/cancel-reservation", cancelReservation)

	return h.ServeHTTPContext
}

func rpcAuthedHandler(generator *generator.Config, signer *blocksigner.Signer) chainhttp.HandlerFunc {
	h := httpjson.NewServeMux(writeHTTPError)

	if generator != nil {
		h.HandleFunc("POST", "/rpc/generator/submit", generator.Submit)
		h.HandleFunc("POST", "/rpc/generator/get-blocks", generator.GetBlocks)
	}
	if signer != nil {
		h.HandleFunc("POST", "/rpc/signer/sign-block", signer.SignBlock)
	}

	return h.ServeHTTPContext
}
