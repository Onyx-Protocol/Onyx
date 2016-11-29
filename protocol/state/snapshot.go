// Package state provides types for encapsulating blockchain state.
package state

import (
	"fmt"

	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/patricia"
)

// PriorIssuances maps an "issuance hash" to the time (in Unix millis)
// at which it should expire from the issuance memory.
type PriorIssuances map[bc.Hash]uint64

// Snapshot encompasses a snapshot of entire blockchain state. It
// consists of two patricia state trees (one for asset-v1 outputs, one
// for asset-v2) and the issuances memory.
type Snapshot struct {
	Tree1, Tree2 *patricia.Tree
	Issuances    PriorIssuances
}

func (s *Snapshot) Insert(o *Output) error {
	var tree *patricia.Tree
	switch o.TypedOutput.(type) {
	case *bc.Outputv1:
		tree = s.Tree1
	case *bc.Outputv2:
		tree = s.Tree2
	default:
		return fmt.Errorf("unknown output type %T", o.TypedOutput)
	}
	return tree.Insert(OutputTreeItem(o))
}

func (s *Snapshot) Delete(key []byte) error {
	err := s.Tree1.Delete(key)
	if err != nil {
		return err
	}
	return s.Tree2.Delete(key)
}

func (s *Snapshot) ContainsKey(bkey []byte, version uint64) bool {
	switch version {
	case 0:
		return s.Tree1.ContainsKey(bkey) || s.Tree2.ContainsKey(bkey)
	case 1:
		return s.Tree1.ContainsKey(bkey)
	case 2:
		return s.Tree2.ContainsKey(bkey)
	}
	return false
}

func (s *Snapshot) Contains(bkey, val []byte, version uint64) bool {
	switch version {
	case 0: // "unknown"
		return s.Tree1.Contains(bkey, val) || s.Tree2.Contains(bkey, val)
	case 1:
		return s.Tree1.Contains(bkey, val)
	case 2:
		return s.Tree2.Contains(bkey, val)
	}
	return false
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
		Tree1:     patricia.Copy(original.Tree1),
		Tree2:     patricia.Copy(original.Tree2),
		Issuances: make(PriorIssuances, len(original.Issuances)),
	}
	for k, v := range original.Issuances {
		c.Issuances[k] = v
	}
	return c
}

// Empty returns an empty state snapshot.
func Empty() *Snapshot {
	return &Snapshot{
		Tree1:     new(patricia.Tree),
		Tree2:     new(patricia.Tree),
		Issuances: make(PriorIssuances),
	}
}
