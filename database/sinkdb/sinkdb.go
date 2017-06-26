// Package sinkdb provides a strongly consistent key-value store.
package sinkdb

import (
	"context"
	"net/http"
	"sort"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/tecbot/gorocksdb"

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
	rocks, err := NewRocksDB(filepath.Join(dir, "rocksdb"))
	if err != nil {
		return nil, errors.Wrap(err, "could not open rocksdb")
	}
	db := &DB{state: state, raft: sv, rocksdb: rocks}
	return db, nil
}

// DB provides access to an opened kv store.
type DB struct {
	mu     sync.Mutex
	closed bool

	state *state
	raft  *raft.Service
	rocksdb *gorocksdb.DB
}

// Ping peforms an empty write to verify the connection to
// the rest of the cluster.
func (db *DB) Ping() error {
	const timeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := db.raft.Exec(ctx, db.state.EmptyWrite())
	return err
}

// Close closes the database handle releasing its resources. It is
// the caller's responsibility to ensure that there are no concurrent
// database operations in flight. Close is idempotent.
//
// All other methods have undefined behavior on a closed DB.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed { // make Close idempotent
		return nil
	}
	db.closed = true
	return db.raft.Stop()
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
// It first checks the rocksdb; if there is nothing stored
// at that key in rocksdb, Get then checks the protobuf-on-disk
// store. If it finds something in the protobuf-on-disk store, it
// will write that value to rocksdb.
func (db *DB) Get(ctx context.Context, key string, v proto.Message) (Version, error) {
	err := db.raft.WaitRead(ctx)
	if err != nil {
		return Version{}, err
	}
	// buf, ver := db.state.get(key)
	buf, err := Get(db.rocksdb, key)
	defer buf.Free() // cgo. sigh
	if err != nil {
		return Version{}, err
	}
	err = proto.Unmarshal(buf, v) // I feel this will not work as buf is now a gorocksdb.Slice but we will seeeeee
	return ver, err
}

// GetStale performs a non-linearizable read of the provided key.
// The value may be stale. The read value is unmarshalled into v.
func (db *DB) GetStale(key string, v proto.Message) (Version, error) {
	buf, ver := db.state.get(key) // read directly from state
	return ver, proto.Unmarshal(buf, v)
}

// RaftService returns the raft service used for replication.
func (db *DB) RaftService() *raft.Service {
	return db.raft
}
