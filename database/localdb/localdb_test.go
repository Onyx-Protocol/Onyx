package localdb

import (
	"io/ioutil"
	"os"
	"testing"
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

	err = ldb1.Put("foo", []byte("bar"))
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

	value, err := ldb2.Get("foo")
	if string(value) != "bar" {
		t.Fatalf("expected value read to be 'bar', got %s", value)
	}
}
