package core

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"chain/core/accesstoken"
	"chain/database/pg/pgtest"
	"chain/database/raft"
)

func TestCreatGrantValidation(t *testing.T) {
	ctx := context.Background()
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)

	accessTokens := &accesstoken.CredentialStore{db}
	_, err := accessTokens.Create(ctx, "test-token", "")
	if err != nil {
		t.Fatal(err)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	raftDir := filepath.Join(currentDir, "/.testraft")
	err = os.Mkdir(raftDir, 0700)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(raftDir)

	raftDB, err := raft.Start("", raftDir, "", new(http.Client), false)
	if err != nil {
		t.Fatal(err)
	}

	api := &API{
		mux:          http.NewServeMux(),
		raftDB:       raftDB,
		accessTokens: accessTokens,
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

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	raftDir := filepath.Join(currentDir, "/.testraft")
	err = os.Mkdir(raftDir, 0700)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(raftDir)

	raftDB, err := raft.Start("", raftDir, "", new(http.Client), false)
	if err != nil {
		t.Fatal(err)
	}

	api := &API{
		mux:          http.NewServeMux(),
		raftDB:       raftDB,
		accessTokens: accessTokens,
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

		grants, ok := gs["items"].([]apiGrant)
		if ok {
			if len(grants) != 1-i {
				t.Error("expected grant to get deleted instead saw ", len(grants), 1-i)
			}
			continue
		}

		// also have to do this check, for when there are 0 grants left
		grants2, ok := gs["items"].([]struct{})
		if ok {
			if len(grants2) != 1-i {
				t.Error("expected grant to get deleted instead saw ", len(grants), 1-i)
			}
			continue
		}

		t.Error("could not convert grant response")
	}

}
