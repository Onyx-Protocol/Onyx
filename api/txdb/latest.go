package txdb

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/database/pg"
	"chain/errors"
	"chain/net/trace/span"
)

// LatestBlock returns the most recent block.  It is not an error (at
// this layer) to have an empty blocks table.
func (s *Store) LatestBlock(ctx context.Context) (*bc.Block, error) {
	s.latestBlockCache.mutex.Lock()
	defer s.latestBlockCache.mutex.Unlock()

	if result := s.latestBlockCache.block; result != nil {
		return result, nil
	}

	// Fall back to the database, keep the cache locked.

	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	const q = `SELECT data FROM blocks ORDER BY height DESC LIMIT 1`
	b := new(bc.Block)
	err := pg.QueryRow(ctx, q).Scan(b)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}

	s.setLatestBlockCache(b, nil, true)

	return b, nil
}

// setLatestValidBlock stores the given block as the head of the
// blockchain.  It also wakes up any threads waiting in
// waitForNewValidBlock.
func (s *Store) setLatestBlockCache(b *bc.Block, stateTree *patricia.Tree, cacheLocked bool) {
	if !cacheLocked {
		s.latestBlockCache.mutex.Lock()
		defer s.latestBlockCache.mutex.Unlock()
	}

	// TODO(kr): get a signal from the underlying storage (postgres)
	// when another process has landed a block and we should
	// invalidate this cache.
	s.latestBlockCache.block = b
	s.latestBlockCache.stateTree = stateTree
}
