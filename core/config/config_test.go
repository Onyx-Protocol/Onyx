package config

import (
	"chain/database/pg/pgtest"
	"chain/database/sinkdb"
	"chain/database/sinkdb/sinkdbtest"
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"
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

	c, _ = Load(ctx, db, sdb)
	if c != nil {
		t.Errorf("Expected nil config")
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
