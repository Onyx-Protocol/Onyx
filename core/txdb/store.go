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

	initialBlockHash bc.Hash
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

	if block.Height == 1 {
		s.initialBlockHash = block.Hash()
	}

	// Note: this is done last so that callers of LatestBlock
	// can safely assume the block they get has been applied.
	s.setLatestBlockCache(block, false)

	return nil
}

// SaveStateTree saves a state tree snapshot to the database.
func (s *Store) SaveStateTree(ctx context.Context, height uint64, tree *patricia.Tree) error {
	err := storeStateTreeSnapshot(ctx, s.db, tree, height)
	return errors.Wrap(err, "saving state tree")
}

// LatestStateTree returns the most recent state tree stored in
// the database and its corresponding block height.
func (s *Store) LatestStateTree(ctx context.Context) (*patricia.Tree, uint64, error) {
	return getStateTreeSnapshot(ctx, s.db)
}

func (s *Store) FinalizeBlock(ctx context.Context, height uint64) error {
	_, err := s.db.Exec(ctx, `SELECT pg_notify('newblock', $1)`, height)
	return err
}

func (s *Store) InitialBlockHash(ctx context.Context) (bc.Hash, error) {
	if s.initialBlockHash == (bc.Hash{}) {
		// Calling LatestBlock is a simple way to block until there's at
		// least one block in the blockchain.
		b, err := s.LatestBlock(ctx)
		if err != nil {
			return bc.Hash{}, err
		}
		if b.Height == 1 {
			s.initialBlockHash = b.Hash()
		} else {
			err = s.db.QueryRow(ctx, "SELECT block_hash FROM blocks WHERE height = 1").Scan(&s.initialBlockHash)
			if err != nil {
				return bc.Hash{}, err
			}
		}
	}
	return s.initialBlockHash, nil
}
