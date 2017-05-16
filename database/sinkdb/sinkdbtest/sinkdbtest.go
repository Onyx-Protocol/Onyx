package sinkdbtest

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"chain/database/sinkdb"
)

// NewDB creates a new sinkdb instance with a random temporary
// storage directory.
func NewDB(t testing.TB) (sdb *sinkdb.DB, cleanup func()) {
	tempDir, err := ioutil.TempDir("", "chain-syncdbtest")
	if err != nil {
		t.Fatal(err)
	}
	sdb, err = sinkdb.Open("", tempDir, "", new(http.Client), false)
	if err != nil {
		t.Fatal(err)
	}

	// TODO(jackson): support closing sinkdb.DB and stopping
	// any goroutines spawned in the raft package.
	cleanup = func() { os.RemoveAll(tempDir) }

	return sdb, cleanup
}
