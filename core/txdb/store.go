package txdb

import (
	"sync"

	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/cos/state"
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
		mutex     sync.Mutex
		block     *bc.Block
		stateTree *patricia.Tree
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

func (s *Store) ApplyBlock(
	ctx context.Context,
	block *bc.Block,
	addedUTXOs []*state.Output,
	removedUTXOs []*state.Output,
	assets map[bc.AssetID]*state.AssetState,
	state *patricia.Tree,
) ([]*bc.Tx, error) {
	dbtx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	ctx = pg.NewContext(ctx, dbtx)
	defer dbtx.Rollback(ctx)

	newHashes, err := insertBlock(ctx, dbtx, block)
	if err != nil {
		return nil, errors.Wrap(err, "insert block")
	}

	newMap := make(map[bc.Hash]bool, len(newHashes))
	newTxs := make([]*bc.Tx, 0, len(newHashes))
	oldTxs := make([]*bc.Tx, 0, len(block.Transactions)-len(newHashes))
	for _, hash := range newHashes {
		newMap[hash] = true
	}
	for _, tx := range block.Transactions {
		if newMap[tx.Hash] {
			newTxs = append(newTxs, tx)
			continue
		}
		oldTxs = append(oldTxs, tx)
	}

	err = insertAssetDefinitionPointers(ctx, dbtx, assets)
	if err != nil {
		return nil, errors.Wrap(err, "insert ADPs")
	}

	err = insertAssetDefinitions(ctx, dbtx, block)
	if err != nil {
		return nil, errors.Wrap(err, "writing asset definitions")
	}

	err = addIssuances(ctx, dbtx, assets, true)
	if err != nil {
		return nil, errors.Wrap(err, "adding issuances")
	}

	err = writeStateTree(ctx, dbtx, state)
	if err != nil {
		return nil, errors.Wrap(err, "updating state tree")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "committing db transaction")
	}

	// Note: this is done last so that callers of LatestBlock
	// can safely assume the block they get has been applied.
	s.setLatestBlockCache(block, patricia.Copy(state), false)

	return newTxs, nil
}

// StateTree returns the state tree of the latest block.
// It takes the height of the expected block, so that it can
// return an error if the height does not match, preventing
// race conditions.
func (s *Store) StateTree(ctx context.Context, block uint64) (*patricia.Tree, error) {
	s.latestBlockCache.mutex.Lock()
	defer s.latestBlockCache.mutex.Unlock()

	if block != 0 && (s.latestBlockCache.block == nil || s.latestBlockCache.block.Height != block) {
		return nil, cos.ErrBadStateHeight
	}

	if s.latestBlockCache.stateTree == nil {
		stateTree, err := stateTree(ctx, s.db)
		if err != nil {
			return nil, err
		}

		s.setLatestBlockCache(s.latestBlockCache.block, stateTree, true)
	}
	return patricia.Copy(s.latestBlockCache.stateTree), nil
}

func (s *Store) FinalizeBlock(ctx context.Context, height uint64) error {
	_, err := s.db.Exec(ctx, `SELECT pg_notify('newblock', $1)`, height)
	return err
}
