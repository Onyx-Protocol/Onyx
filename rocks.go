package main

import (
	"fmt"
	"log"

	"github.com/tecbot/gorocksdb"
)

func main() {
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	db, err := gorocksdb.OpenDb(opts, "db")
	if err != nil {
		log.Fatal(err)
	}

	ro := gorocksdb.NewDefaultReadOptions()
	value, err := db.Get(ro, []byte("foo"))
	defer value.Free() // tktk do this after error check?
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("value: %s\n", value.Data())

	wo := gorocksdb.NewDefaultWriteOptions()

	err = db.Put(wo, []byte("foo"), []byte("bar"))
	if err != nil {
		log.Fatal(err)
	}
	value, err = db.Get(ro, []byte("foo"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("value: %s\n", value.Data())
}
