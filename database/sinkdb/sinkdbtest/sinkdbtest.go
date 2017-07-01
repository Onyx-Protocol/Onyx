package sinkdbtest

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"chain/database/sinkdb"
)

const dataDirectoryPrefix = `chain-syncdbtest`

// NewDB creates a new sinkdb instance with a random temporary
// storage directory and a new single-node raft cluster.
func NewDB(t testing.TB) *sinkdb.DB {
	gcDataDirectories() // clean up old data directories from previous tests

	tempDir, err := ioutil.TempDir("", dataDirectoryPrefix)
	if err != nil {
		t.Fatal(err)
	}
	sdb, err := sinkdb.Open("", tempDir, new(http.Client), &testStore{tempDir})
	if err != nil {
		t.Fatal(err)
	}
	err = sdb.RaftService().Init()
	if err != nil {
		t.Fatal(err)
	}

	// set a finalizer to close the DB to reclaim file descriptors, etc.
	runtime.SetFinalizer(sdb, (*sinkdb.DB).Close)
	return sdb
}

func gcDataDirectories() {
	tempDir := os.TempDir()
	cutoff := time.Now().Add(-time.Hour * 24)
	dirents, _ := ioutil.ReadDir(tempDir)
	for _, dirent := range dirents {
		if !strings.HasPrefix(dirent.Name(), dataDirectoryPrefix) {
			continue
		}
		if dirent.ModTime().After(cutoff) {
			continue
		}
		os.RemoveAll(filepath.Join(tempDir, dirent.Name()))
	}
}

// fulfills the sinkdb.Store interface
type testStore struct{}

func (ts *testStore) Put(name string, value []byte) error {
	return ioutil.WriteFile(name, value, 0666)
}

func (ts *testStore) Get(name string) ([]byte, error) {
	return ioutil.ReadFile(name)
}
