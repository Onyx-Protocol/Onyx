package config

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"chain/database/pg/pgtest"
	"chain/database/sinkdb"
	"chain/database/sinkdb/sinkdbtest"
	"chain/errors"
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

// newTestConfig returns a new Config object
// which has an ID, but no other fields set
func newTestConfig(t *testing.T) *Config {
	c := new(Config)
	b := make([]byte, 10)
	rand.Read(b)
	c.Id = hex.EncodeToString(b)
	return c
}
