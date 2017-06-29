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
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic detecting stale config")
		}
	}()

	ctx := context.Background()
	sdb := sinkdbtest.NewDB(t)
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	c := newTestConfig(t)

	// Write config to sinkdb
	sdb.Exec(ctx,
		sinkdb.IfNotExists("/core/config"),
		sinkdb.Set("/core/config", c),
	)

	Load(ctx, db, sdb)
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
