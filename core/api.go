// Package core provides http handlers for all Chain operations.
package core

import (
	"strconv"
	"time"

	"chain/core/appdb"
	"chain/core/explorer"
	"chain/core/generator"
	"chain/core/signer"
	"chain/core/txdb"
	chainhttp "chain/net/http"
	"chain/net/http/httpjson"
	"chain/net/http/pat"
)

const (
	sessionTokenLifetime = 2 * 7 * 24 * time.Hour
	defAccountPageSize   = 100
	defBalancePageSize   = 100
	defAssetPageSize     = 100
	defGenericPageSize   = 100
)

// Handler returns a handler that serves the Chain HTTP API. Param nouserSecret
// will be used as the password for routes starting with /nouser/.
func Handler(nouserSecret string, generatorConfig *generator.Config, signer *signer.Signer, store *txdb.Store, pool *txdb.Pool, explorer *explorer.Explorer) chainhttp.Handler {
	h := pat.New()
	a := &api{
		store:     store,
		pool:      pool,
		explorer:  explorer,
		generator: generatorConfig,
	}

	pwHandler := httpjson.NewServeMux(writeHTTPError)
	pwHandler.HandleFunc("POST", "/v3/login", login)
	h.AddFunc("POST", "/v3/login", userCredsAuthn(pwHandler.ServeHTTPContext))

	nouserHandler := chainhttp.HandlerFunc(nouserAuthn(nouserSecret, nouserHandler()))
	h.Add("GET", "/nouser/", nouserHandler)
	h.Add("PUT", "/nouser/", nouserHandler)
	h.Add("POST", "/nouser/", nouserHandler)
	h.Add("DELETE", "/nouser/", nouserHandler)

	tokenHandler := chainhttp.HandlerFunc(tokenAuthn(a.tokenAuthedHandler()))
	h.Add("GET", "/", tokenHandler)
	h.Add("PUT", "/", tokenHandler)
	h.Add("POST", "/", tokenHandler)
	h.Add("DELETE", "/", tokenHandler)

	rpcHandler := chainhttp.HandlerFunc(rpcAuthn(rpcAuthedHandler(generatorConfig, signer)))
	h.Add("GET", "/rpc/", rpcHandler)
	h.Add("PUT", "/rpc/", rpcHandler)
	h.Add("POST", "/rpc/", rpcHandler)
	h.Add("DELETE", "/rpc/", rpcHandler)

	return h
}

func nouserHandler() chainhttp.HandlerFunc {
	h := httpjson.NewServeMux(writeHTTPError)

	// These routes must trust the client to enforce access control.
	// Think twice before adding something here.
	h.HandleFunc("GET", "/nouser/invitations/:invID", appdb.GetInvitation)
	h.HandleFunc("POST", "/nouser/invitations/:invID/create-user", createUserFromInvitation)
	h.HandleFunc("POST", "/nouser/password-reset/start", startPasswordReset)
	h.HandleFunc("POST", "/nouser/password-reset/check", checkPasswordReset)
	h.HandleFunc("POST", "/nouser/password-reset/finish", finishPasswordReset)

	return h.ServeHTTPContext
}

type api struct {
	store     *txdb.Store
	pool      *txdb.Pool
	explorer  *explorer.Explorer
	generator *generator.Config
}

func (a *api) tokenAuthedHandler() chainhttp.HandlerFunc {
	h := httpjson.NewServeMux(writeHTTPError)
	h.HandleFunc("GET", "/v3/projects", listProjects)
	h.HandleFunc("POST", "/v3/projects", createProject)
	h.HandleFunc("GET", "/v3/projects/:projID", getProject)
	h.HandleFunc("PUT", "/v3/projects/:projID", updateProject)
	h.HandleFunc("DELETE", "/v3/projects/:projID", archiveProject)
	h.HandleFunc("POST", "/v3/invitations", createInvitation)
	h.HandleFunc("GET", "/v3/projects/:projID/admin-node/summary", a.getAdminNodeSummary)
	h.HandleFunc("POST", "/v3/projects/:projID/manager-nodes", createManagerNode)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID/accounts", listAccounts)
	h.HandleFunc("POST", "/v3/manager-nodes/:mnodeID/accounts", createAccount)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID/assets/:assetID/balances", listAccountsWithAsset) // EXPERIMENTAL - implemented for Glitterco
	h.HandleFunc("GET", "/v3/projects/:projID/issuer-nodes", listIssuerNodes)
	h.HandleFunc("POST", "/v3/projects/:projID/issuer-nodes", createIssuerNode)
	h.HandleFunc("GET", "/v3/issuer-nodes/:inodeID", getIssuerNode)
	h.HandleFunc("PUT", "/v3/issuer-nodes/:inodeID", updateIssuerNode)
	h.HandleFunc("DELETE", "/v3/issuer-nodes/:inodeID", archiveIssuerNode)
	h.HandleFunc("GET", "/v3/issuer-nodes/:inodeID/assets", a.listAssets)
	h.HandleFunc("POST", "/v3/issuer-nodes/:inodeID/assets", a.createAsset)
	h.HandleFunc("GET", "/v3/accounts/:accountID", getAccount)
	h.HandleFunc("GET", "/v3/accounts/:accountID/balance", a.accountBalance)
	h.HandleFunc("POST", "/v3/accounts/:accountID/addresses", createAddr)
	h.HandleFunc("POST", "/v3/accounts/:accountID/utxos", listAccountUTXOs)
	h.HandleFunc("PUT", "/v3/accounts/:accountID", updateAccount)
	h.HandleFunc("DELETE", "/v3/accounts/:accountID", archiveAccount)
	h.HandleFunc("GET", "/v3/assets/:assetID", a.getIssuerAsset)
	h.HandleFunc("PUT", "/v3/assets/:assetID", updateAsset)
	h.HandleFunc("DELETE", "/v3/assets/:assetID", archiveAsset)
	h.HandleFunc("POST", "/v3/assets/:assetID/issue", issueAsset) // DEPRECATED
	h.HandleFunc("POST", "/v3/transact/build", build)
	h.HandleFunc("POST", "/v3/transact/submit", submit)
	h.HandleFunc("POST", "/v3/transact/finalize", submitSingle) // DEPRECATED
	h.HandleFunc("POST", "/v3/transact/finalize-batch", submit) // DEPRECATED
	h.HandleFunc("POST", "/v3/transact/cancel-reservation", cancelReservation)
	h.HandleFunc("GET", "/v3/user", getAuthdUser)
	h.HandleFunc("POST", "/v3/user/email", updateUserEmail)
	h.HandleFunc("POST", "/v3/user/password", updateUserPassword)
	h.HandleFunc("PUT", "/v3/user/:userID/role", updateUserRole)
	h.HandleFunc("GET", "/v3/users", listUsers)
	h.HandleFunc("GET", "/v3/authcheck", func() {})
	h.HandleFunc("GET", "/v3/api-tokens", listAPITokens)
	h.HandleFunc("POST", "/v3/api-tokens", createAPIToken)
	h.HandleFunc("DELETE", "/v3/api-tokens/:tokenID", appdb.DeleteAuthToken)

	// Auditor node endpoints -- DEPRECATED: use explorer endpoints instead
	h.HandleFunc("GET", "/v3/auditor/blocks", a.listBlocks)
	h.HandleFunc("GET", "/v3/auditor/blocks/:blockID/summary", a.getBlockSummary)
	h.HandleFunc("GET", "/v3/auditor/transactions/:txID", a.getTx)
	h.HandleFunc("GET", "/v3/auditor/assets/:assetID", a.getAsset)
	h.HandleFunc("GET", "/v3/auditor/assets/:assetID/utxos", a.listExplorerUTXOsByAsset)
	h.HandleFunc("POST", "/v3/auditor/get-assets", a.getExplorerAssets) // EXPERIMENTAL(jeffomatic), implemented for R3 demo

	// Explorer node endpoints
	h.HandleFunc("GET", "/v3/explorer/blocks", a.listBlocks)
	h.HandleFunc("GET", "/v3/explorer/blocks/:blockID/summary", a.getBlockSummary)
	h.HandleFunc("GET", "/v3/explorer/transactions/:txID", a.getTx)
	h.HandleFunc("GET", "/v3/explorer/assets/:assetID", a.getAsset)
	h.HandleFunc("GET", "/v3/explorer/assets/:assetID/utxos", a.listExplorerUTXOsByAsset)
	h.HandleFunc("POST", "/v3/explorer/get-assets", a.getExplorerAssets) // EXPERIMENTAL(jeffomatic), implemented for R3 demo

	// Orderbook endpoints
	h.HandleFunc("POST", "/v3/contracts/orderbook", findOrders)
	h.HandleFunc("POST", "/v3/contracts/orderbook/:accountID", findAccountOrders)

	// Voting system endpoints
	h.HandleFunc("GET", "/v3/accounts/:accountID/voting-rights", findAccountVotingRights)
	h.HandleFunc("GET", "/v3/contracts/voting-rights/:assetID/owners", getVotingRightOwners)
	h.HandleFunc("POST", "/v3/contracts/voting-tokens/tally", getVotingTokenTally)
	h.HandleFunc("POST", "/v3/contracts/voting-tokens/votes", getVotingTokenVotes)

	return h.ServeHTTPContext
}

func rpcAuthedHandler(generator *generator.Config, signer *signer.Signer) chainhttp.HandlerFunc {
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

// For time query-params that can be in either RFC3339 or
// Unix-timestamp form.
func parseTime(s string) (t time.Time, err error) {
	t, err = time.Parse(time.RFC3339, s)
	if err != nil {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return t, err
		}
		t = time.Unix(i, 0)
	}
	return t, nil
}
