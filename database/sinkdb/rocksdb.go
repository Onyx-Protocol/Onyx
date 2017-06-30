package sinkdb

// import (
// 	"github.com/tecbot/gorocksdb"
// )

// make state fulfill the (net/raft).State interface
// this is in the rocks file so we could use a build flag to build a different
// Write implementation
func (s *state) Write(key string, value []byte) error {
	return s.db.Put(key, value)
}

// func (s *state) Get(key string) (*gorocksdb.Slice, Version, error) {
// 	slice, err := s.db.Get(key)
// 	_, ok := s.state[key]
// 	n := s.version[key]
// 	return slice, Version{key, ok, n}, err
// }
