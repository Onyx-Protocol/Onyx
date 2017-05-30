package sinkdb

// Version records the version of a particular key
// when it is read.
// Every time a key is set, its version changes.
type Version struct {
	key string
	n   uint64 // raft log index
}

// Exists returns whether v's key exists
// when it was read.
func (v Version) Exists() bool {
	return v.n != 0
}

// Key returns the key for which v is valid.
func (v Version) Key() string {
	return v.key
}
