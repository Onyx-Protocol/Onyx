// Package state provides types for encapsulating blockchain state.
package state

import (
	"chain/protocol/bc"
	"chain/protocol/patricia"
)

// PriorIssuances maps an "issuance hash" to the time (in Unix millis)
// at which it should expire from the issuance memory.
type PriorIssuances map[bc.Hash]uint64

// Snapshot encompasses a snapshot of entire blockchain state. It
// consists of a patricia state tree and the issuances memory.
type Snapshot struct {
	B1Hash    bc.Hash
	Tree      *patricia.Tree
	Issuances PriorIssuances
}

// PruneIssuances modifies a Snapshot, removing all issuance hashes
// with expiration times earlier than the provided timestamp.
func (s *Snapshot) PruneIssuances(timestampMS uint64) {
	for hash, expiryMS := range s.Issuances {
		if timestampMS > expiryMS {
			delete(s.Issuances, hash)
		}
	}
}

// Copy makes a copy of provided snapshot. Copying a snapshot is an
// O(n) operation where n is the number of issuance hashes in the
// snapshot's issuance memory.
func Copy(original *Snapshot) *Snapshot {
	// TODO(kr): consider making type Snapshot truly immutable.
	// We already handle it that way in many places (with explicit
	// calls to Copy to get the right behavior).
	c := &Snapshot{
		B1Hash:    original.B1Hash,
		Tree:      patricia.Copy(original.Tree),
		Issuances: make(PriorIssuances, len(original.Issuances)),
	}
	for k, v := range original.Issuances {
		c.Issuances[k] = v
	}
	return c
}

// Empty returns an empty state snapshot.
func Empty(b1Hash bc.Hash) *Snapshot {
	return &Snapshot{
		B1Hash:    b1Hash,
		Tree:      new(patricia.Tree),
		Issuances: make(PriorIssuances),
	}
}
