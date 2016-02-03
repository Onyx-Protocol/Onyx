package txdb

import (
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type Store struct{}

var _ fedchain.Store = (*Store)(nil)

func (s *Store) ApplyTx(ctx context.Context, tx *bc.Tx) error {
	err := InsertTx(ctx, tx)
	if err != nil {
		return errors.Wrap(err, "insert into txs")
	}

	err = InsertPoolTx(ctx, tx)
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
	err = InsertPoolOutputs(ctx, outputs)
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
	err = InsertPoolInputs(ctx, deleted)
	return errors.Wrap(err, "insert into pool inputs")
}

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
	return errors.Wrap(err, "delete from pool_inputs")
}

func (s *Store) PoolTxs(ctx context.Context) ([]*bc.Tx, error) {
	return PoolTxs(ctx)
}

func (s *Store) NewPoolViewForPrevouts(ctx context.Context, txs []*bc.Tx) (state.ViewReader, error) {
	return NewPoolViewForPrevouts(ctx, txs)
}

func (s *Store) ApplyBlock(ctx context.Context, block *bc.Block, adps map[bc.AssetID]*bc.AssetDefinitionPointer, delta []*state.Output) ([]*bc.Tx, error) {
	newHashes, err := InsertBlock(ctx, block)
	if err != nil {
		return nil, errors.Wrap(err, "insert block")
	}

	newMap := make(map[bc.Hash]bool, len(newHashes))
	newTxs := make([]*bc.Tx, 0, len(newHashes))
	for _, hash := range newHashes {
		newMap[hash] = true
	}
	for _, tx := range block.Transactions {
		if newTx := newMap[tx.Hash]; newTx {
			newTxs = append(newTxs, tx)
		}
	}

	err = InsertAssetDefinitionPointers(ctx, adps)
	if err != nil {
		return nil, errors.Wrap(err, "insert ADPs")
	}

	err = InsertAssetDefinitions(ctx, block)
	if err != nil {
		return nil, errors.Wrap(err, "writing asset definitions")
	}

	err = RemoveBlockSpentOutputs(ctx, delta)
	if err != nil {
		return nil, errors.Wrap(err, "remove block spent outputs")
	}

	err = InsertBlockOutputs(ctx, delta)
	if err != nil {
		return nil, errors.Wrap(err, "insert block outputs")
	}

	return newTxs, nil
}

func (s *Store) LatestBlock(ctx context.Context) (*bc.Block, error) {
	return LatestBlock(ctx)
}

func (s *Store) NewViewForPrevouts(ctx context.Context, txs []*bc.Tx) (state.ViewReader, error) {
	return NewViewForPrevouts(ctx, txs)
}
