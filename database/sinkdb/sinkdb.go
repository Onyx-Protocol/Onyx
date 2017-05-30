// Package sinkdb provides a strongly consistent key-value store.
package sinkdb

import (
	"context"
	"net/http"
	"sort"

	"github.com/golang/protobuf/proto"

	"chain/database/sinkdb/internal/sinkpb"
	"chain/errors"
	"chain/net/raft"
)

// ErrConflict is returned by Exec when an instruction was
// not completed because its preconditions were not met.
var ErrConflict = errors.New("transaction conflict")

// Open initializes the key-value store and returns a database handle.
func Open(laddr, dir string, httpClient *http.Client) (*DB, error) {
	state := newState()
	sv, err := raft.Start(laddr, dir, httpClient, state)
	if err != nil {
		return nil, err
	}
	db := &DB{state: state, raft: sv}
	return db, nil
}

// DB provides access to an opened kv store.
type DB struct {
	state *state
	raft  *raft.Service
}

// Exec executes the provided operations
// after combining them with All.
func (db *DB) Exec(ctx context.Context, ops ...Op) error {
	all := All(ops...)
	if all.err != nil {
		return all.err
	}

	// Disallow multiple writes to the same key.
	sort.Slice(all.effects, func(i, j int) bool {
		return all.effects[i].Key < all.effects[j].Key
	})
	var lastKey string
	for _, e := range all.effects {
		if e.Key == lastKey {
			err := errors.New("duplicate write")
			return errors.Wrap(err, e.Key)
		}
		lastKey = e.Key
	}

	encoded, err := proto.Marshal(&sinkpb.Instruction{
		Conditions: all.conds,
		Operations: all.effects,
	})
	if err != nil {
		return err
	}
	satisfied, err := db.raft.Exec(ctx, encoded)
	if err != nil {
		return err
	}
	if !satisfied {
		return ErrConflict
	}
	return nil
}

// Get performs a linearizable read of the provided key. The
// read value is unmarshalled into v.
func (db *DB) Get(ctx context.Context, key string, v proto.Message) (found bool, err error) {
	err = db.raft.WaitRead(ctx)
	if err != nil {
		return false, err
	}
	buf, found := db.state.get(key)
	return found, proto.Unmarshal(buf, v)
}

// GetStale performs a non-linearizable read of the provided key.
// The value may be stale. The read value is unmarshalled into v.
func (db *DB) GetStale(key string, v proto.Message) (found bool, err error) {
	buf, found := db.state.get(key) // read directly from state
	return found, proto.Unmarshal(buf, v)
}

// RaftService returns the raft service used for replication.
func (db *DB) RaftService() *raft.Service {
	return db.raft
}
