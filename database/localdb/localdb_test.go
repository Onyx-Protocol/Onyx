package localdb

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/tecbot/gorocksdb"

	"chain/database/localdb/internal/localdbtest"
)

func TestRestartDB(t *testing.T) {
	rocksDir, err := ioutil.TempDir("", "rocks_testdb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(rocksDir)

	// Create a new fresh db and write... something.
	ldb1, err := Open(rocksDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ldb1.Close()

	testItem := &localdbtest.TestItem{Value: "bar"}
	err = ldb1.Put("foo", testItem)
	if err != nil {
		t.Fatal(err)
	}

	ldb1.Close()

	// Re-open the database and verify that the write is still there.
	ldb2, err := Open(rocksDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ldb2.Close()

	resultItem := new(localdbtest.TestItem)
	err = ldb2.Get("foo", resultItem)
	if err != nil {
		t.Fatal(err)
	}

	if resultItem.Value != "bar" {
		t.Fatalf("expected value read to be 'bar', got %s", resultItem.Value)
	}
}

// Make err-nil helper function
func TestDirPrefix(t *testing.T) {
	db := newTestDb(t, func(opts *gorocksdb.Options) {
		opts.SetPrefixExtractor(&dirPrefixTransform{})
	})
	defer db.Close()

	testItem := &localdbtest.TestItem{Value: "bar"}
	err := db.Put("/foo", testItem)
	if err != nil {
		t.Fatal(err)
	}
	testItem1 := &localdbtest.TestItem{Value: "bar1"}
	err = db.Put("/foo/bar", testItem1)
	if err != nil {
		t.Fatal(err)
	}
	testItem2 := &localdbtest.TestItem{Value: "bar2"}
	err = db.Put("/foo/baz", testItem2)
	if err != nil {
		t.Fatal(err)
	}
	testItem3 := &localdbtest.TestItem{Value: "err"}
	err = db.Put("/bar", testItem3)
	if err != nil {
		t.Fatal(err)
	}

	iter := db.store.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer iter.Close()
	prefix := []byte("/foo")
	numFound := 0
	for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
		numFound++
	}
	if iter.Err() != nil {
		t.Fatal(iter.Err())
	}
	if numFound != 3 {
		t.Fatal("Incorrect number found: expected 3, got ", numFound)
	}
}

func newTestDb(t *testing.T, applyOpts func(opts *gorocksdb.Options)) *DB {
	opts := gorocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	if applyOpts != nil {
		applyOpts(opts)
	}
	dir, err := ioutil.TempDir("", "rocks_testdb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	rocks, err := gorocksdb.OpenDb(opts, dir)
	if err != nil {
		t.Fatal(err)
	}
	return &DB{store: rocks}
}
