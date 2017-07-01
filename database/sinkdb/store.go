package sinkdb

import "github.com/tecbot/gorocksdb"

type Store interface {
	Get(name string) ([]byte, error)
	Put(name string, value []byte) error
}

// fulfills the Store interface
type rocksStore struct {
	db *gorocksdb.DB
}

func (rs *rocksStore) Put(name string, value []byte) error {
	wo := gorocksdb.NewDefaultWriteOptions()
	return rs.db.Put(wo, []byte(name), value)
}

func (rs *rocksStore) Get(name string) ([]byte, error) {
	ro := gorocksdb.NewDefaultReadOptions()
	slice, err := rs.db.Get(ro, []byte(name))
	defer slice.Free()
	if err != nil {
		return []byte{}, err
	}
	return slice.Data(), nil
}

func NewStore(rocksDir string) (Store, error) {
	// TODO(tessr): tune rocksdb
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	rocks, err := gorocksdb.OpenDb(opts, rocksDir)
	if err != nil {
		return nil, err
	}

	return &rocksStore{db: rocks}, nil
}
