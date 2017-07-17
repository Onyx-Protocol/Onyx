package config

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"chain/core/accesstoken"
	"chain/database/pg/pgtest"
	"chain/database/sinkdb"
	"chain/database/sinkdb/sinkdbtest"
	"chain/errors"
	"chain/net/http/authz"
	"chain/protocol/bc"
)

func TestDetectStaleConfig(t *testing.T) {
	ctx := context.Background()
	sdb := sinkdbtest.NewDB(t)
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	c := newTestConfig(t)

	// Write config to sinkdb
	sdb.Exec(ctx,
		sinkdb.Set("/core/config", c),
	)

	var err error
	c, err = Load(ctx, db, sdb)
	if c != nil {
		t.Errorf("Expected nil config")
	}
	err = errors.Root(err)
	if err != ErrStaleRaftConfig {
		t.Errorf("Expected ErrStaleRaftConfig")
	}
}

func TestLoadUnconfigured(t *testing.T) {
	ctx := context.Background()
	sdb := sinkdbtest.NewDB(t)
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	c := newTestConfig(t)

	var err error
	c, err = Load(ctx, db, sdb)
	if c != nil {
		t.Errorf("Expected nil config")
	}
	must(t, err)
}

func TestLoadConfigNoErr(t *testing.T) {
	ctx := context.Background()
	sdb := sinkdbtest.NewDB(t)
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	c := newTestConfig(t)

	// Write config to sinkdb and pg
	sdb.Exec(ctx,
		sinkdb.Set("/core/config", c),
	)
	const q = `INSERT INTO core_id (id) VALUES ($1)`
	var err error
	_, err = db.ExecContext(ctx, q, c.Id)
	must(t, err)

	c, err = Load(ctx, db, sdb)
	if c == nil {
		t.Errorf("Expected loaded config")
	}
	must(t, err)
}

func TestMigrateAccessTokens(t *testing.T) {
	ctx := context.Background()
	sdb := sinkdbtest.NewDB(t)
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)

	// Saving test config to PG
	const q = `INSERT INTO config (id, blockchain_id, configured_at, is_signer, is_generator, max_issuance_window_ms) 
				VALUES ($1, $2, $3, $4, $5, $6)`
	var blockchainID bc.Hash
	blockchainID.UnmarshalText([]byte("test-blockchain"))
	_, err := db.ExecContext(ctx, q, "test-id", &blockchainID, time.Now(), true, true, bc.DurationMillis(24*time.Hour))
	must(t, err)

	cs := &accesstoken.CredentialStore{DB: db}
	var pgClientToken *accesstoken.Token
	var pgNetworkToken *accesstoken.Token
	pgClientToken, err = cs.Create(ctx, "a", "client")
	pgNetworkToken, err = cs.Create(ctx, "b", "network")
	must(t, err)

	var c *Config
	c, err = Load(ctx, db, sdb)
	must(t, err)
	if c == nil {
		t.Errorf("Expected loaded config")
	}

	// Read access tokens from sinkDB
	var Policies = []string{
		"client-readwrite",
		"crosscore",
	}
	var clientToken authz.Grant
	var networkToken authz.Grant
	for _, p := range Policies {
		var accessTokens []authz.Grant
		var grantList authz.GrantList
		_, err = sdb.Get(ctx, GrantPrefix+p, &grantList)
		must(t, err)
		for _, g := range grantList.Grants {
			accessTokens = append(accessTokens, *g)
		}
		if p == "client-readwrite" {
			clientToken = accessTokens[0]
		} else if p == "crosscore" {
			networkToken = accessTokens[0]
		}
	}

	// Check that sinkDB access tokens equal PG access tokens
	var clientData []byte
	var networkData []byte
	clientData, err = json.Marshal(map[string]interface{}{"id": "a"})
	must(t, err)
	networkData, err = json.Marshal(map[string]interface{}{"id": "b"})
	must(t, err)
	if !bytes.Equal(clientToken.GuardData, clientData) {
		t.Errorf("Guard data incorrect")
	}
	if clientToken.GuardType != "access_token" {
		t.Errorf("Guard type incorrect: wanted %q, got %q", "access_token", clientToken.GuardType)
	}
	if clientToken.CreatedAt != pgClientToken.Created.Format(time.RFC3339) {
		t.Errorf("Time created incorrect: wanted %q, got %q", pgClientToken.Created.Format(time.RFC3339), clientToken.CreatedAt)
	}
	if !bytes.Equal(networkToken.GuardData, networkData) {
		t.Errorf("Guard data incorrect")
	}
	if networkToken.GuardType != "access_token" {
		t.Errorf("Guard type incorrect: wanted %q, got %q", "access_token", networkToken.GuardType)
	}
	if networkToken.CreatedAt != pgNetworkToken.Created.Format(time.RFC3339) {
		t.Errorf("Time created incorrect: wanted %q, got %q", pgNetworkToken.Created.Format(time.RFC3339), networkToken.CreatedAt)
	}
}

// newTestConfig returns a new Config object
// which has an ID, but no other fields set
func newTestConfig(t *testing.T) *Config {
	c := new(Config)
	b := make([]byte, 10)
	rand.Read(b)
	c.Id = hex.EncodeToString(b)
	return c
}
