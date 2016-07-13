package txdb

import (
	"sync"

	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
)

// A Store encapsulates storage for blockchain validation.
// It satisfies the interface cos.Store, and provides additional
// methods for querying current and historical data.
type Store struct {
	db *sql.DB

	latestBlockCache struct {
		mutex sync.Mutex
		block *bc.Block
	}
	latestStateTreeCache struct {
		mutex  sync.Mutex
		height uint64
		tree   *patricia.Tree
	}
}

var _ cos.Store = (*Store)(nil)

// NewStore creates and returns a new Store object.
//
// A Store manages its own database transactions, so
// it requires a handle to a SQL database.
// For testing purposes, it is usually much faster
// and more convenient to use package chain/cos/memstore
// instead.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetTxs looks up transactions in the blockchain by their hashes.
func (s *Store) GetTxs(ctx context.Context, hashes ...bc.Hash) (bcTxs map[bc.Hash]*bc.Tx, err error) {
	return getBlockchainTxs(ctx, s.db, hashes...)
}

// SaveBlock persists a new block in the database.
func (s *Store) SaveBlock(ctx context.Context, block *bc.Block) error {
	dbtx, err := s.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err)
	}
	ctx = pg.NewContext(ctx, dbtx)
	defer dbtx.Rollback(ctx)

	err = insertBlock(ctx, dbtx, block)
	if err != nil {
		return errors.Wrap(err, "insert block")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return errors.Wrap(err, "committing db transaction")
	}

	s.setLatestBlockCache(block, false)
	return nil
}

// SaveStateTree saves a state tree snapshot to the database.
func (s *Store) SaveStateTree(ctx context.Context, height uint64, tree *patricia.Tree) error {
	err := storeStateTreeSnapshot(ctx, s.db, tree, height)
	if err != nil {
		return errors.Wrap(err, "saving state tree")
	}
	s.setLatestStateTreeCache(patricia.Copy(tree), height, false)
	return nil
}

// StateTree returns the state tree of the block at the provided height.
// It will cache the most recently requested state tree, which should be
// the most recent one.
func (s *Store) StateTree(ctx context.Context, height uint64) (*patricia.Tree, error) {
	s.latestStateTreeCache.mutex.Lock()
	defer s.latestStateTreeCache.mutex.Unlock()

	if s.latestStateTreeCache.tree == nil || s.latestStateTreeCache.height != height {
		tree, err := getStateTreeSnapshot(ctx, pg.FromContext(ctx), height)
		if err != nil {
			return nil, err
		}
		s.setLatestStateTreeCache(tree, height, true)
	}
	return patricia.Copy(s.latestStateTreeCache.tree), nil
}

func (s *Store) FinalizeBlock(ctx context.Context, height uint64) error {
	_, err := s.db.Exec(ctx, `SELECT pg_notify('newblock', $1)`, height)
	return err
}
