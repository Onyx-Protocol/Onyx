package state

import (
	"fmt"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/patricia"
)

// Snapshot encompasses a snapshot of entire blockchain state. It
// consists of a patricia state tree and the nonce set.
//
// Nonces maps a nonce entry's ID to the time (in Unix millis)
// at which it should expire from the nonce set.
//
// Snapshot satisfies the bc.BlockchainState interface.
type Snapshot struct {
	Tree   *patricia.Tree
	Nonces map[bc.Hash]uint64
}

func (s *Snapshot) AddNonce(id bc.Hash, expiryMS uint64) error {
	if s.Nonces[id] >= expiryMS {
		return fmt.Errorf("conflicting nonce %x", id[:])
	}
	s.Nonces[id] = expiryMS
	return nil
}

func (s *Snapshot) DeleteSpentOutput(id bc.Hash) error {
	if !s.Tree.Contains(id[:]) {
		return fmt.Errorf("invalid prevout %x", id[:])
	}
	s.Tree.Delete(id[:])
	return nil
}

func (s *Snapshot) AddOutput(id bc.Hash) error {
	return s.Tree.Insert(id[:])
}

// PruneNonces modifies a Snapshot, removing all nonce IDs
// with expiration times earlier than the provided timestamp.
func (s *Snapshot) PruneNonces(timestampMS uint64) {
	for hash, expiryMS := range s.Nonces {
		if timestampMS > expiryMS {
			delete(s.Nonces, hash)
		}
	}
}

// Copy makes a copy of provided snapshot. Copying a snapshot is an
// O(n) operation where n is the number of nonces in the snapshot's
// nonce set.
func (s *Snapshot) Copy() *Snapshot {
	// TODO(kr): consider making type Snapshot truly immutable.
	// We already handle it that way in many places (with explicit
	// calls to Copy to get the right behavior).
	c := &Snapshot{
		Tree:   new(patricia.Tree),
		Nonces: make(map[bc.Hash]uint64, len(s.Nonces)),
	}
	*c.Tree = *s.Tree
	for k, v := range s.Nonces {
		c.Nonces[k] = v
	}
	return c
}

// Empty returns an empty state snapshot.
func Empty() *Snapshot {
	return &Snapshot{
		Tree:   new(patricia.Tree),
		Nonces: make(map[bc.Hash]uint64),
	}
}

// ApplyBlock updates s in place.
func (s *Snapshot) ApplyBlock(block *bc.BlockEntries) error {
	s.PruneNonces(block.Body.TimestampMS)
	for i, tx := range block.Transactions {
		err := tx.Apply(s)
		if err != nil {
			return errors.Wrapf(err, "applying block transaction %d", i)
		}
	}
	return nil
}
