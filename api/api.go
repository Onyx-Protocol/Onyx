// Package api provides http handlers for all Chain operations.
package api

import (
	"time"

	"chain/api/appdb"
	chainhttp "chain/net/http"
	"chain/net/http/httpjson"
	"chain/net/http/pat"
)

const (
	sessionTokenLifetime = 2 * 7 * 24 * time.Hour
	defActivityPageSize  = 50
	defBucketPageSize    = 100
	defBalancePageSize   = 100
	defAssetPageSize     = 100
)

// Handler returns a handler that serves the Chain HTTP API. Param nouserSecret
// will be used as the password for routes starting with /nouser/.
func Handler(nouserSecret string) chainhttp.Handler {
	h := chainhttp.PatServeMux{PatternServeMux: pat.New()}

	pwHandler := httpjson.NewServeMux(writeHTTPError)
	pwHandler.HandleFunc("POST", "/v3/login", login)
	h.AddFunc("POST", "/v3/login", userCredsAuthn(pwHandler.ServeHTTPContext))

	nouserHandler := chainhttp.HandlerFunc(nouserAuthn(nouserSecret, nouserHandler()))
	h.Add("GET", "/nouser/", nouserHandler)
	h.Add("PUT", "/nouser/", nouserHandler)
	h.Add("POST", "/nouser/", nouserHandler)
	h.Add("DELETE", "/nouser/", nouserHandler)

	tokenHandler := chainhttp.HandlerFunc(tokenAuthn(tokenAuthedHandler()))
	h.Add("GET", "/", tokenHandler)
	h.Add("PUT", "/", tokenHandler)
	h.Add("POST", "/", tokenHandler)
	h.Add("DELETE", "/", tokenHandler)

	return h
}

func nouserHandler() chainhttp.HandlerFunc {
	h := httpjson.NewServeMux(writeHTTPError)

	// These routes must trust the client to enforce access control.
	// Think twice before adding something here.
	h.HandleFunc("GET", "/nouser/invitations/:invID", appdb.GetInvitation)
	h.HandleFunc("POST", "/nouser/invitations/:invID/create-user", createUserFromInvitation)
	h.HandleFunc("POST", "/nouser/invitations/:invID/add-existing", addMemberFromInvitation)
	h.HandleFunc("POST", "/nouser/password-reset/start", startPasswordReset)
	h.HandleFunc("POST", "/nouser/password-reset/finish", finishPasswordReset)

	return h.ServeHTTPContext
}

func tokenAuthedHandler() chainhttp.HandlerFunc {
	h := httpjson.NewServeMux(writeHTTPError)
	h.HandleFunc("GET", "/v3/projects", listApplications)
	h.HandleFunc("POST", "/v3/projects", createApplication)
	h.HandleFunc("GET", "/v3/projects/:projID", appdb.GetApplication)
	h.HandleFunc("PUT", "/v3/projects/:projID", updateApplication)
	h.HandleFunc("POST", "/v3/projects/:projID/invitations", createInvitation)
	h.HandleFunc("GET", "/v3/projects/:projID/members", appdb.ListMembers)
	h.HandleFunc("POST", "/v3/projects/:projID/members", addMember)
	h.HandleFunc("PUT", "/v3/projects/:projID/members/:userID", updateMember)
	h.HandleFunc("DELETE", "/v3/projects/:projID/members/:userID", appdb.RemoveMember)
	h.HandleFunc("GET", "/v3/projects/:projID/manager-nodes", appdb.ListWallets)
	h.HandleFunc("POST", "/v3/projects/:projID/manager-nodes", createWallet)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID", appdb.GetWallet)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID/accounts", listBuckets)
	h.HandleFunc("POST", "/v3/manager-nodes/:mnodeID/accounts", createBucket)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID/balance", walletBalance)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID/activity", getWalletActivity)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID/transactions/:txID", appdb.WalletTxActivity)
	h.HandleFunc("GET", "/v3/projects/:projID/issuer-nodes", appdb.ListAssetGroups)
	h.HandleFunc("POST", "/v3/projects/:projID/issuer-nodes", createAssetGroup)
	h.HandleFunc("GET", "/v3/issuer-nodes/:inodeID", appdb.GetAssetGroup)
	h.HandleFunc("GET", "/v3/issuer-nodes/:inodeID/assets", listAssets)
	h.HandleFunc("POST", "/v3/issuer-nodes/:inodeID/assets", createAsset)
	h.HandleFunc("GET", "/v3/issuer-nodes/:inodeID/activity", getAssetGroupActivity)
	h.HandleFunc("GET", "/v3/accounts/:accountID/balance", bucketBalance)
	h.HandleFunc("GET", "/v3/accounts/:accountID/activity", getBucketActivity)
	h.HandleFunc("POST", "/v3/accounts/:accountID/addresses", createAddr)
	h.HandleFunc("GET", "/v3/assets/:assetID", appdb.GetAsset)
	h.HandleFunc("GET", "/v3/assets/:assetID/activity", getAssetActivity)
	h.HandleFunc("POST", "/v3/assets/:assetID/issue", issueAsset)
	h.HandleFunc("POST", "/v3/transact/build", build)
	h.HandleFunc("POST", "/v3/transact/transfer", buildSingle)
	h.HandleFunc("POST", "/v3/transact/trade", buildSingle)
	h.HandleFunc("POST", "/v3/transact/finalize", walletFinalize)
	h.HandleFunc("POST", "/v3/transact/finalize-batch", batchFinalize)
	h.HandleFunc("POST", "/v3/transact/cancel-reservation", cancelReservation)
	h.HandleFunc("GET", "/v3/user", getAuthdUser)
	h.HandleFunc("POST", "/v3/user/email", updateUserEmail)
	h.HandleFunc("POST", "/v3/user/password", updateUserPassword)
	h.HandleFunc("GET", "/v3/authcheck", func() {})
	h.HandleFunc("GET", "/v3/api-tokens", listAPITokens)
	h.HandleFunc("POST", "/v3/api-tokens", createAPIToken)
	h.HandleFunc("DELETE", "/v3/api-tokens/:tokenID", appdb.DeleteAuthToken)
	return h.ServeHTTPContext
}
