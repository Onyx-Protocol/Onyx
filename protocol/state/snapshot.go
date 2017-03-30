package state

import (
	"fmt"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/patricia"
)

// Snapshot encompasses a snapshot of entire blockchain state. It
// consists of a patricia state tree and the issuances memory.
//
// Issuances maps an "issuance hash" to the time (in Unix millis)
// at which it should expire from the issuance memory.
//
// TODO(bobg): replace the issuances memory with a nonce set per the
// latest spec (deferred from
// https://github.com/chain/chain/pull/788).
//
// TODO: consider making type Snapshot truly immutable.  We already
// handle it that way in many places (with explicit calls to Copy to
// get the right behavior).  PruneNonces and the Apply functions would
// have to produce new Snapshots rather than updating Snapshots in
// place.
type Snapshot struct {
	Tree      *patricia.Tree
	Issuances map[bc.Hash]uint64
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
	c := &Snapshot{
		Tree:      new(patricia.Tree),
		Issuances: make(map[bc.Hash]uint64, len(original.Issuances)),
	}
	*c.Tree = *original.Tree
	for k, v := range original.Issuances {
		c.Issuances[k] = v
	}
	return c
}

// Empty returns an empty state snapshot.
func Empty() *Snapshot {
	return &Snapshot{
		Tree:      new(patricia.Tree),
		Issuances: make(map[bc.Hash]uint64),
	}
}

// ApplyBlock updates s in place.
func (s *Snapshot) ApplyBlock(block *bc.BlockEntries) error {
	s.PruneIssuances(block.Body.TimestampMS)
	for i, tx := range block.Transactions {
		err := s.ApplyTx(tx)
		if err != nil {
			return errors.Wrapf(err, "applying block transaction %d", i)
		}
	}
	return nil
}

// ApplyTx updates s in place.
func (s *Snapshot) ApplyTx(tx *bc.TxEntries) error {
	for _, issID := range tx.IssuanceIDs {
		// Add new issuances. They must not conflict with issuances already
		// present.
		if s.Issuances[issID] >= tx.Body.MaxTimeMS {
			return fmt.Errorf("conflicting issuance %x", issID[:])
		}
		s.Issuances[issID] = tx.Body.MaxTimeMS
	}

	// Remove spent outputs. Each output must be present.
	for _, prevout := range tx.SpentOutputIDs {
		if !s.Tree.Contains(prevout[:]) {
			return fmt.Errorf("invalid prevout %x", prevout[:])
		}
		s.Tree.Delete(prevout[:])
	}

	// Add new outputs. They must not yet be present.
	for _, o := range tx.OutputIDs {
		err := s.Tree.Insert(o[:])
		if err != nil {
			return err
		}
	}
	return nil
}
