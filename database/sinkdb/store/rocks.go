package store

import (
	"fmt"
	"log"

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

// 	ro := gorocksdb.NewDefaultReadOptions()
// 	value, err := db.Get(ro, []byte("foo"))
// 	defer value.Free()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Printf("value: %s\n", value.Data())

// 	wo := gorocksdb.NewDefaultWriteOptions()

// 	err = db.Put(wo, []byte("foo"), []byte("bar"))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	value, err = db.Get(ro, []byte("foo"))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Printf("value: %s\n", value.Data())
// }


func Put(db *gorocksdb.DB, key string, value []byte) error {
	// TODO(tessr): tune rocksdb
	wo := gorocksdb.NewDefaultReadOptions()
	return db.Put(wo, []byte(key), value)
}

func Get(db *gorocksdb.DB, key string) ([]byte, error) {
	// TODO(tessr): tune rocksdb
	ro := gorocksdb.NewDefaultReadOptions()
	return db.Get(ro, []byte(key))
}