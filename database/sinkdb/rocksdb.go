package sinkdb

import (
	"github.com/tecbot/gorocksdb"
)

func NewRocksDB(datadir string) (*gorocksdb.DB, error) {
	// TODO(tessr): tune rocksdb
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)

	return gorocksdb.OpenDb(opts, datadir)
}

// make state fulfill the (net/raft).State interface
// this is in the rocks file so we could use a build flag to build a different
// Write implementation
func (s *state) Write(key string, value []byte) error {
	// TODO(tessr): tune rocksdb - tweak write options
	wo := gorocksdb.NewDefaultWriteOptions()
	return s.db.Put(wo, []byte(key), value)
}

func (s *state) Get(key string) (*gorocksdb.Slice, Version, error) {
	// TODO(tessr): tune rocksdb - tweak read options
	ro := gorocksdb.NewDefaultReadOptions()
	slice, err := s.db.Get(ro, []byte(key))
	_, ok := s.state[key]
	n := s.version[key]
	return slice, Version{key, ok, n}, err
}
