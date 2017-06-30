package sinkdb

// TODO(tessr): build flags to specify rocksdb

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/tecbot/gorocksdb"
)

type store struct {
	db *gorocksdb.DB
}

func NewStore(datadir string) (*store, error) {
	// TODO(tessr): tune rocksdb
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)

	db, err := gorocksdb.OpenDb(opts, datadir)
	if err != nil {
		return nil, err
	}

	return &store{db: db}, nil
}

func (s *store) Get(ctx context.Context, key string, v proto.Message) (Version, error) {
	// TODO(tessr): tune rocksdb - tweak read options
	ro := gorocksdb.NewDefaultReadOptions()
	slice, err := s.db.Get(ro, []byte(key))
	defer slice.Free()
	if err != nil {
		return
	}

	err = proto.Unmarshal(slice.Data(), v)
	if err != nil {
		return err
		_, ok := s.state[key]
	}

	n := s.version[key]
	return slice, Version{key, ok, n}, err
}

func (s *store) Put(key string, value []byte) error {
	// TODO(tessr): tune rocksdb - tweak write options
	wo := gorocksdb.NewDefaultWriteOptions()
	return s.db.Put(wo, []byte(key), value)
}
