package core

import (
	"context"
	"net/http"
	"testing"

	"chain/core/accesstoken"
	"chain/database/pg/pgtest"
	"chain/database/sinkdb/sinkdbtest"
	"chain/net/http/authz"
)

func TestCreatGrantValidation(t *testing.T) {
	ctx := context.Background()
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)

	accessTokens := &accesstoken.CredentialStore{db}
	_, err := accessTokens.Create(ctx, "test-token", "")
	if err != nil {
		t.Fatal(err)
	}

	sdb := sinkdbtest.NewDB(t)
	api := &API{
		mux:          http.NewServeMux(),
		sdb:          sdb,
		accessTokens: accessTokens,
		grants:       authz.NewStore(sdb, GrantPrefix),
	}

	validCases := []apiGrant{
		{
			GuardType: "access_token",
			GuardData: map[string]interface{}{
				"id": "test-token",
			},
			Policy: "client-readwrite",
		},
		{
			GuardType: "x509",
			GuardData: map[string]interface{}{
				"subject": map[string]interface{}{
					"CN": "should-work",
				},
			},
			Policy: "client-readwrite",
		},
	}

	for i, c := range validCases {
		_, err := api.createGrant(ctx, c)
		if err != nil {
			t.Errorf("valid grant %d (%v) error: %v", i, c, err)
		}
	}

	errCases := []apiGrant{
		// blank guard type
		{
			GuardType: "",
		},

		// unrecognized guard type
		{
			GuardType: "invalid",
		},

		// unknown token
		{
			GuardType: "access_token",
			GuardData: map[string]interface{}{
				"id": "invalid-token",
			},
			Policy: "client-readwrite",
		},

		// invalid token data
		{
			GuardType: "access_token",
			GuardData: map[string]interface{}{
				"invalid": "invalid",
			},
			Policy: "client-readwrite",
		},

		// invalid policy
		{
			GuardType: "access_token",
			GuardData: map[string]interface{}{
				"id": "invalid-token",
			},
			Policy: "invalid",
		},

		// invalid x509 subject
		{
			GuardType: "x509",
			GuardData: map[string]interface{}{
				"subject": map[string]interface{}{
					"invalid": "invalid",
				},
			},
			Policy: "client-readwrite",
		},

		// non-subject x509 attribs
		{
			GuardType: "x509",
			GuardData: map[string]interface{}{
				"subject": map[string]interface{}{
					"CN": "valid-cn",
				},
				"invalid": "invalid",
			},
			Policy: "client-readwrite",
		},
	}

	for i, c := range errCases {
		_, err := api.createGrant(ctx, c)
		if err == nil {
			t.Errorf("error grant %d (%v): error was was nil", i, c)
		}
	}
}

func TestDeleteGrants(t *testing.T) {
	ctx := context.Background()
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)

	accessTokens := &accesstoken.CredentialStore{db}
	_, err := accessTokens.Create(ctx, "test-token", "")
	if err != nil {
		t.Fatal(err)
	}

	sdb := sinkdbtest.NewDB(t)
	api := &API{
		mux:          http.NewServeMux(),
		sdb:          sdb,
		accessTokens: accessTokens,
		grants:       authz.NewStore(sdb, GrantPrefix),
	}

	fixture := []apiGrant{
		{
			GuardType: "access_token",
			GuardData: map[string]interface{}{
				"id": "test-token",
			},
			Policy: "client-readwrite",
		},
		{
			GuardType: "x509",
			GuardData: map[string]interface{}{
				"subject": map[string]interface{}{
					"CN": "should-work",
				},
			},
			Policy: "client-readwrite",
		},
	}

	for _, c := range fixture {
		api.createGrant(ctx, c)
	}

	failureCases := []apiGrant{
		{
			GuardType: "access_token",
			GuardData: map[string]interface{}{
				"id": "NOPE",
			},
			Policy: "client-readwrite",
		},
		{
			GuardType: "x509",
			GuardData: map[string]interface{}{
				"subject": map[string]interface{}{
					"CN": "NOPE",
				},
			},
			Policy: "client-readwrite",
		},
	}

	for _, c := range failureCases {
		err = api.deleteGrant(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		gs, err := api.listGrants(ctx)
		if err != nil {
			t.Fatal(err)
		}

		grants, ok := gs["items"].([]apiGrant)
		if ok {
			if len(grants) != len(fixture) {
				t.Error("deletion to fail")
			}
		} else {
			t.Error("could not convert grant response")
		}
	}

	for i, c := range fixture {
		err = api.deleteGrant(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		gs, err := api.listGrants(ctx)
		if err != nil {
			t.Fatal(err)
		}

		length, ok := checkGrantListLength(t, gs, 1-i)
		if !ok {
			t.Errorf("expected grant to get deleted; instead saw %d grants", length)
		}
	}
}

func TestDeleteGrantsByAccessToken(t *testing.T) {
	ctx := context.Background()
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)

	accessTokens := &accesstoken.CredentialStore{db}
	_, err := accessTokens.Create(ctx, "test-token-0", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = accessTokens.Create(ctx, "test-token-1", "")
	if err != nil {
		t.Fatal(err)
	}

	sdb := sinkdbtest.NewDB(t)
	api := &API{
		mux:          http.NewServeMux(),
		sdb:          sdb,
		accessTokens: accessTokens,
		grants:       authz.NewStore(sdb, GrantPrefix),
	}

	// fixture data includes four grants:
	// - two have access token `test-token-0`
	// - one has access token `test-token-1`
	// - one has an x509 cert, and shouldn't get deleted in this test
	fixture := []apiGrant{
		{
			GuardType: "access_token",
			GuardData: map[string]interface{}{
				"id": "test-token-0",
			},
			Policy: "client-readwrite",
		},
		{
			GuardType: "access_token",
			GuardData: map[string]interface{}{
				"id": "test-token-0",
			},
			Policy: "internal",
		},
		{
			GuardType: "access_token",
			GuardData: map[string]interface{}{
				"id": "test-token-1",
			},
			Policy: "client-readwrite",
		},
		{
			GuardType: "x509",
			GuardData: map[string]interface{}{
				"subject": map[string]interface{}{
					"CN": "not-an-access-token",
				},
			},
			Policy: "client-readwrite",
		},
	}

	for _, c := range fixture {
		api.createGrant(ctx, c)
	}

	// first check that we can delete a single grant
	err = api.sdb.Exec(ctx, api.deleteGrantsByAccessToken("test-token-1"))
	if err != nil {
		t.Fatal(err)
	}
	grants, err := api.listGrants(ctx)
	if err != nil {
		t.Fatal(err)
	}
	length, ok := checkGrantListLength(t, grants, 3)
	if !ok {
		t.Fatalf("expected grant list to be length %d, got length %d", 3, length)
	}

	// next check on deleting an access token associates with multiple grants
	err = api.sdb.Exec(ctx, api.deleteGrantsByAccessToken("test-token-0"))
	if err != nil {
		t.Fatal(err)
	}
	grants, err = api.listGrants(ctx)
	if err != nil {
		t.Fatal(err)
	}
	length, ok = checkGrantListLength(t, grants, 1)
	if !ok {
		t.Fatalf("expected grant list to be length %d, got length %d", 1, length)
	}
}

func checkGrantListLength(t *testing.T, gs map[string]interface{}, length int) (int, bool) {
	grants, ok := gs["items"].([]apiGrant)
	if ok {
		return len(grants), len(grants) == length
	}

	// also have to do this check, for the 0 case
	grants2, ok := gs["items"].([]struct{})
	if ok {
		return len(grants2), len(grants2) == length
	}

	t.Fatal("could not convert grant response")
	return -1, false // should never get here
}
