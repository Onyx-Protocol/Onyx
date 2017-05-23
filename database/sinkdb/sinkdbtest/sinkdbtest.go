package sinkdbtest

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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
	sdb, err := sinkdb.Open("", tempDir, new(http.Client))
	if err != nil {
		t.Fatal(err)
	}
	err = sdb.RaftService().Init()
	if err != nil {
		t.Fatal(err)
	}
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
