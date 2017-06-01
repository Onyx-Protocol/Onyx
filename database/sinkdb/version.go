package sinkdb

// Version records the version of a particular key
// when it is read.
// Every time a key is set, its version changes.
type Version struct {
	key string
	ok  bool
	n   uint64 // raft log index
}

// Exists returns whether v's key exists
// when it was read.
func (v Version) Exists() bool {
	// TODO(jackson): use v.n != 0 once we've backfilled versions
	// for Chain Core 1.2.x snapshots.
	return v.ok
}

// Key returns the key for which v is valid.
func (v Version) Key() string {
	return v.key
}
