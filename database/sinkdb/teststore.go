package sinkdb

import (
	"io/ioutil"
)

// fulfills the Store interface
type testStore struct {
	dir string
}

func (ts *testStore) Put(name string, value []byte) error {
	return ioutil.WriteFile(name, value, 0666)
}

func (ts *testStore) Get(name string) ([]byte, error) {
	return ioutil.ReadFile(name)
}
