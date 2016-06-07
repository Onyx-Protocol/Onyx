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

// GetTxs looks up transactions by their hashes
// in the block chain and in the pool.
func (s *Store) GetTxs(ctx context.Context, hashes ...bc.Hash) (poolTxs, bcTxs map[bc.Hash]*bc.Tx, err error) {
	return getTxs(ctx, s.db, hashes...)
}

// ApplyTx adds tx to the pending pool.
func (s *Store) ApplyTx(ctx context.Context, tx *bc.Tx, assets map[bc.AssetID]*state.AssetState) error {
	dbtx, err := s.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err)
	}
	defer dbtx.Rollback(ctx)

	inserted, err := insertTx(ctx, dbtx, tx)
	if err != nil {
		return errors.Wrap(err, "insert into txs")
	}
	if !inserted {
		// Another SQL transaction already succeeded in applying the tx,
		// so there's no need to do anything else.
		return nil
	}

	err = insertPoolTx(ctx, dbtx, tx)
	if err != nil {
		return errors.Wrap(err, "insert into pool txs")
	}

	var outputs []*Output
	for i, out := range tx.Outputs {
		outputs = append(outputs, &Output{
			Output: state.Output{
				Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(i)},
				TxOutput: *out,
			},
		})
	}
	err = insertPoolOutputs(ctx, dbtx, outputs)
	if err != nil {
		return errors.Wrap(err, "insert into utxos")
	}

	err = addIssuances(ctx, dbtx, assets, false)
	if err != nil {
		return errors.Wrap(err, "adding issuances")
	}

	err = dbtx.Commit(ctx)
	return errors.Wrap(err, "committing database transaction")
}

// GetPoolPrevouts looks up all of the transaction's prevouts in the
// pool and returns any of the outputs that exist in the pool.
func (s *Store) GetPoolPrevouts(ctx context.Context, txs []*bc.Tx) (map[bc.Outpoint]*state.Output, error) {
	dbtx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	ctx = pg.NewContext(ctx, dbtx)
	defer dbtx.Rollback(ctx)

	var prevouts []bc.Outpoint
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			prevouts = append(prevouts, in.Previous)
		}
	}

	outs, err := loadPoolOutputs(ctx, dbtx, prevouts)
	if err != nil {
		return outs, err
	}
	return outs, dbtx.Commit(ctx)
}

// CleanPool removes confirmedTxs and conflictTxs from the pool.
func (s *Store) CleanPool(
	ctx context.Context,
	confirmedTxs, conflictTxs []*bc.Tx,
	assets map[bc.AssetID]*state.AssetState,
) error {
	dbtx, err := s.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err)
	}
	defer dbtx.Rollback(ctx)

	var (
		deleteTxHashes   []string
		conflictTxHashes []string
	)
	for _, tx := range append(confirmedTxs, conflictTxs...) {
		deleteTxHashes = append(deleteTxHashes, tx.Hash.String())
	}
	// TODO(kr): ideally there is no distinction between confirmedTxs
	// and conflictTxs here. We currently need to know the difference,
	// because we mix pool outputs with blockchain outputs in postgres,
	// and this means we have to take extra care not to delete confirmed
	// outputs.
	for _, tx := range conflictTxs {
		conflictTxHashes = append(conflictTxHashes, tx.Hash.String())
	}

	// Delete pool_txs
	const txq = `DELETE FROM pool_txs WHERE tx_hash IN (SELECT unnest($1::text[]))`
	_, err = dbtx.Exec(ctx, txq, pg.Strings(deleteTxHashes))
	if err != nil {
		return errors.Wrap(err, "delete from pool_txs")
	}

	// Delete pool outputs
	const outq = `
		DELETE FROM utxos u WHERE tx_hash IN (SELECT unnest($1::text[]))
	`
	_, err = dbtx.Exec(ctx, outq, pg.Strings(conflictTxHashes))
	if err != nil {
		return errors.Wrap(err, "delete from utxos")
	}

	err = setIssuances(ctx, dbtx, assets)
	if err != nil {
		return errors.Wrap(err, "removing issuances")
	}

	err = dbtx.Commit(ctx)
	return errors.Wrap(err, "pool update dbtx commit")
}

// PoolTxs returns the pooled transactions in topological order.
func (s *Store) PoolTxs(ctx context.Context) ([]*bc.Tx, error) {
	// TODO(jeffomatic) - at some point in the future, will we want to keep this
	// cached in an in-memory pool, a la btcd's TxMemPool?
	return poolTxs(ctx, s.db)
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

	// Note: the order of inserting and removing UTXOs is important,
	// otherwise, we'll fail to remove outputs that were added and spent
	// within the same block.
	err = insertBlockOutputs(ctx, dbtx, addedUTXOs)
	if err != nil {
		return nil, errors.Wrap(err, "insert block outputs")
	}

	err = removeBlockSpentOutputs(ctx, dbtx, removedUTXOs)
	if err != nil {
		return nil, errors.Wrap(err, "remove block spent outputs")
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
