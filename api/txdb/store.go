package txdb

import (
	"sync"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type Store struct {
	latestBlockCache struct {
		mutex sync.Mutex
		block *bc.Block
	}
}

var _ fedchain.Store = (*Store)(nil)

// NewStore creates and returns a new Store object.
func NewStore() *Store {
	return &Store{}
}

// GetTxs looks up transactions by their hashes
// in the block chain and in the pool.
func (s *Store) GetTxs(ctx context.Context, hashes ...bc.Hash) (map[bc.Hash]*bc.Tx, error) {
	txs, err := GetTxs(ctx, hashes...)
	if err != nil {
		return nil, err
	}
	return txs, nil
}

// ApplyTx adds tx to the pending pool.
func (s *Store) ApplyTx(ctx context.Context, tx *bc.Tx) error {
	inserted, err := insertTx(ctx, tx)
	if err != nil {
		return errors.Wrap(err, "insert into txs")
	}
	if !inserted {
		// Another SQL transaction already succeeded in applying the tx,
		// so there's no need to do anything else.
		return nil
	}

	err = insertPoolTx(ctx, tx)
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
	err = insertPoolOutputs(ctx, outputs)
	if err != nil {
		return errors.Wrap(err, "insert into utxos")
	}

	var deleted []bc.Outpoint
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			continue
		}
		deleted = append(deleted, in.Previous)
	}
	err = insertPoolInputs(ctx, deleted)
	if err != nil {
		return errors.Wrap(err, "insert into pool inputs")
	}

	err = addIssuances(ctx, sumIssued(tx), false)
	return errors.Wrap(err, "adding issuances")
}

// RemoveTxs removes confirmedTxs and conflictTxs from the pool.
func (s *Store) RemoveTxs(ctx context.Context, confirmedTxs, conflictTxs []*bc.Tx) error {
	db := pg.FromContext(ctx)

	var (
		deleteTxHashes     []string
		deleteInputHashes  []string
		deleteInputIndexes []uint32
		conflictTxHashes   []string
	)
	for _, tx := range append(confirmedTxs, conflictTxs...) {
		deleteTxHashes = append(deleteTxHashes, tx.Hash.String())
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			deleteInputHashes = append(deleteInputHashes, in.Previous.Hash.String())
			deleteInputIndexes = append(deleteInputIndexes, in.Previous.Index)
		}
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
	_, err := db.Exec(ctx, txq, pg.Strings(deleteTxHashes))
	if err != nil {
		return errors.Wrap(err, "delete from pool_txs")
	}

	// Delete pool outputs
	const outq = `
		DELETE FROM utxos u WHERE tx_hash IN (SELECT unnest($1::text[]))
	`
	_, err = db.Exec(ctx, outq, pg.Strings(conflictTxHashes))
	if err != nil {
		return errors.Wrap(err, "delete from utxos")
	}

	// Delete pool_inputs
	const inq = `
		DELETE FROM pool_inputs
		WHERE (tx_hash, index) IN (
			SELECT unnest($1::text[]), unnest($2::integer[])
		)
	`
	_, err = db.Exec(ctx, inq, pg.Strings(deleteInputHashes), pg.Uint32s(deleteInputIndexes))
	if err != nil {
		return errors.Wrap(err, "delete from pool_inputs")
	}

	err = removeIssuances(ctx, sumIssued(conflictTxs...))
	return errors.Wrap(err, "removing issuances")
}

// PoolTxs returns the pooled transactions in topological order.
func (s *Store) PoolTxs(ctx context.Context) ([]*bc.Tx, error) {
	// TODO(jeffomatic) - at some point in the future, will we want to keep this
	// cached in an in-memory pool, a la btcd's TxMemPool?
	return poolTxs(ctx)
}

// NewPoolViewForPrevouts returns a new state view on the pool
// of unconfirmed transactions.
// It loads the prevouts for transactions in txs;
// all other outputs will be omitted from the view.
func (s *Store) NewPoolViewForPrevouts(ctx context.Context, txs []*bc.Tx) (state.ViewReader, error) {
	return newPoolViewForPrevouts(ctx, txs)
}

func (s *Store) ApplyBlock(
	ctx context.Context,
	block *bc.Block,
	adps map[bc.AssetID]*bc.AssetDefinitionPointer,
	delta []*state.Output,
) ([]*bc.Tx, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback(ctx)

	newHashes, err := insertBlock(ctx, block)
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

	err = insertAssetDefinitionPointers(ctx, adps)
	if err != nil {
		return nil, errors.Wrap(err, "insert ADPs")
	}

	err = insertAssetDefinitions(ctx, block)
	if err != nil {
		return nil, errors.Wrap(err, "writing asset definitions")
	}

	err = removeBlockSpentOutputs(ctx, delta)
	if err != nil {
		return nil, errors.Wrap(err, "remove block spent outputs")
	}

	err = insertBlockOutputs(ctx, delta)
	if err != nil {
		return nil, errors.Wrap(err, "insert block outputs")
	}

	err = addIssuances(ctx, sumIssued(block.Transactions...), true)
	if err != nil {
		return nil, errors.Wrap(err, "adding issuances")
	}

	err = removeIssuances(ctx, sumIssued(oldTxs...))
	if err != nil {
		return nil, errors.Wrap(err, "removing confirmed issuances from pool")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "committing db transaction")
	}

	// Note: this is done last so that callers of LatestBlock
	// can safely assume the block they get has been applied.
	s.setLatestBlockCache(block, false)

	return newTxs, nil
}

// NewViewForPrevouts returns a new state view on the blockchain.
// It loads the prevouts for transactions in txs;
// all other outputs will be omitted from the view.
func (s *Store) NewViewForPrevouts(ctx context.Context, txs []*bc.Tx) (state.ViewReader, error) {
	return newViewForPrevouts(ctx, txs)
}
