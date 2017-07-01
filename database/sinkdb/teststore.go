package sinkdb

import (
	"io/ioutil"
	"path/filepath"
)

// fulfills the Store interface
type testStore struct {
	dir string
}

func (ts *testStore) Put(name string, value []byte) error {
	return ioutil.WriteFile(filepath.Join(ts.dir, name), value, 0666)
}

func (ts *testStore) Get(name string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(ts.dir, name))
}
