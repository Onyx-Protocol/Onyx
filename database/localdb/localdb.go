// Package localdb provides an interface for storing local data.
// Data stored in localdb is not synchronized to any other cored
// processes.
package localdb

import (
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/tecbot/gorocksdb"

	"chain/errors"
)

// DB provides access to a kv store.
type DB struct {
	store *gorocksdb.DB

	mu     sync.Mutex
	closed bool
}

var errDBClosed = errors.New("database is closed")

// Put writes protocol buffer values to the database, at the provided key.
func (db *DB) Put(key string, value proto.Message) error {
	// TODO(tessr): use 'Exec' instead of Put

	if db.closed {
		return errDBClosed
	}

	encodedValue, err := proto.Marshal(value)
	if err != nil {
		return errors.Wrap(err)
	}

	// TODO(tessr): tune rocksdb. assess write options
	wo := gorocksdb.NewDefaultWriteOptions()
	return db.store.Put(wo, []byte(key), encodedValue)
}

// Get fetches the data associated with the provided key and
// unmarshals it into the provided protocol buffer.
func (db *DB) Get(key string, v proto.Message) error {
	if db.closed {
		return errDBClosed
	}

	// TODO(tessr): tune rocksdb. assess read options
	ro := gorocksdb.NewDefaultReadOptions()
	slice, err := db.store.Get(ro, []byte(key))
	if err != nil {
		return errors.Wrap(err)
	}
	defer slice.Free()
	return proto.Unmarshal(slice.Data(), v)
}

// Open opens a new localDB, using the provided dataDir as
// the data directory for RocksDB.
func Open(dataDir string) (*DB, error) {
	// TODO(tessr): tune rocksdb. assess all these options
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	rocks, err := gorocksdb.OpenDb(opts, dataDir)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &DB{store: rocks}, nil
}

// Close closes the database. The database must be opened
// again before it will accept reads or writes.
func (db *DB) Close() {
	db.mu.Lock()
	defer db.mu.Unlock()
	if !db.closed {
		db.store.Close()
		db.closed = true
	}
}
