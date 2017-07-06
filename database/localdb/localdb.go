package localdb

import (
	"chain/errors"

	"github.com/golang/protobuf/proto"
	"github.com/tecbot/gorocksdb"
)

// DB provides access to a kv store.
type DB struct {
	store  *gorocksdb.DB
	closed bool
}

// TODO(tessr): use 'Exec' instead of Put
func (db *DB) Put(key string, value proto.Message) error {
	encodedValue, err := proto.Marshal(value)
	if err != nil {
		return errors.Wrap(err)
	}

	// TODO(tessr): tune rocksdb. assess write options
	wo := gorocksdb.NewDefaultWriteOptions()
	return db.store.Put(wo, []byte(key), encodedValue)
}

func (db *DB) Get(key string, v proto.Message) error {
	// TODO(tessr): tune rocksdb. assess read options
	ro := gorocksdb.NewDefaultReadOptions()
	slice, err := db.store.Get(ro, []byte(key))
	defer slice.Free()
	if err != nil {
		return errors.Wrap(err)
	}
	return proto.Unmarshal(slice.Data(), v)
}

func Open(rocksDir string) (*DB, error) {
	// TODO(tessr): tune rocksdb. assess all these options
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	rocks, err := gorocksdb.OpenDb(opts, rocksDir)
	if err != nil {
		return nil, err
	}

	return &DB{store: rocks}, nil
}

func (db *DB) Close() {
	if !db.closed {
		db.store.Close()
		db.closed = true
	}
}
