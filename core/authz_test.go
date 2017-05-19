package core

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chain/core/accesstoken"
	"chain/database/pg/pgtest"
	"chain/database/sinkdb/sinkdbtest"
	"chain/net/http/authz"
)

func TestAuthz(t *testing.T) {
	ctx := context.Background()
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	accessTokens := &accesstoken.CredentialStore{db}
	sdb := sinkdbtest.NewDB(t)

	mux := http.NewServeMux()
	mux.Handle("/raft/", sdb.RaftService())

	var handler http.Handler = mux
	handler = AuthHandler(handler, sdb, accessTokens, nil, nil)

	api := &API{
		mux:          http.NewServeMux(),
		sdb:          sdb,
		accessTokens: accessTokens,
		grants:       authz.NewStore(sdb, GrantPrefix),
	}
	api.buildHandler()
	mux.Handle("/", api)
	server := httptest.NewServer(handler)
	defer server.Close()

	testPolicies := []string{
		"client-readwrite",
		"client-readonly",
		"crosscore",
		"crosscore-signblock",
		"monitoring",
		"internal",
		"public",
	}
	tokens := make(map[string]*accesstoken.Token)
	for i := 0; i < len(testPolicies); i++ {
		token, err := accessTokens.Create(ctx, fmt.Sprintf("token%d", i), "")
		if err != nil {
			t.Fatal(err)
		}
		tokens[Policies[i]] = token
		_, err = api.createGrant(ctx, apiGrant{
			GuardType: "access_token",
			GuardData: map[string]interface{}{"id": token.ID},
			Policy:    Policies[i],
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	pathTokenMap := map[string]map[string]bool{
		"/create-account": map[string]bool{
			"client-readwrite":    true,
			"client-readonly":     false,
			"crosscore":           false,
			"crosscore-signblock": false,
			"monitoring":          false,
			"internal":            false,
			"public":              false,
		},
		"/list-accounts": map[string]bool{
			"client-readwrite":    true,
			"client-readonly":     true,
			"crosscore":           false,
			"crosscore-signblock": false,
			"monitoring":          false,
			"internal":            false,
			"public":              false,
		},
		"/reset": map[string]bool{
			"client-readwrite":    true,
			"client-readonly":     false,
			"crosscore":           false,
			"crosscore-signblock": false,
			"monitoring":          false,
			"internal":            true,
			"public":              false,
		},
		crosscoreRPCPrefix + "get-block": map[string]bool{
			"client-readwrite":    false,
			"client-readonly":     false,
			"crosscore":           true,
			"crosscore-signblock": true,
			"monitoring":          false,
			"internal":            false,
			"public":              false,
		},
		crosscoreRPCPrefix + "signer/sign-block": map[string]bool{
			"client-readwrite":    false,
			"client-readonly":     false,
			"crosscore":           false,
			"crosscore-signblock": true,
			"monitoring":          false,
			"internal":            true,
			"public":              false,
		},
		"/info": map[string]bool{
			"client-readwrite":    true,
			"client-readonly":     true,
			"crosscore":           true,
			"crosscore-signblock": true,
			"monitoring":          true,
			"internal":            true,
			"public":              false,
		},
		"/debug/pprof/symbol": map[string]bool{
			"client-readwrite":    true,
			"client-readonly":     true,
			"crosscore":           false,
			"crosscore-signblock": false,
			"monitoring":          true,
			"internal":            false,
			"public":              false,
		},
		"/raft/msg": map[string]bool{
			"client-readwrite":    false,
			"client-readonly":     false,
			"crosscore":           false,
			"crosscore-signblock": false,
			"monitoring":          false,
			"internal":            true,
			"public":              false,
		},
		"/dashboard": map[string]bool{ // public is open to all
			"client-readwrite":    true,
			"client-readonly":     true,
			"crosscore":           true,
			"crosscore-signblock": true,
			"monitoring":          true,
			"internal":            true,
			"public":              true,
		},
	}

	for path, policyMap := range pathTokenMap {
		for policy, want := range policyMap {
			got := tryRPC(t, server.URL, path, tokens[policy])
			if got != want {
				t.Errorf("auth(%s, %s) = %t want %t", path, policy, got, want)
			}
		}
	}
}

func tryRPC(t testing.TB, baseURL, path string, token *accesstoken.Token) bool {
	req, err := http.NewRequest("POST", baseURL+path, bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	req.SetBasicAuth(token.ID, strings.Split(token.Token, ":")[1])

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	resp.Body.Close()

	if resp.StatusCode == 500 {
		t.Fatal("unexpected 500 error")
	}

	return resp.StatusCode != http.StatusForbidden
}
