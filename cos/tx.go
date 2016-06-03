package cos

import (
	"time"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/cos/validation"
	"chain/errors"
	"chain/metrics"
)

// AddTx inserts tx into the set of "pending" transactions available
// to be included in the next block produced by GenerateBlock.
//
// It validates tx against the blockchain state and the existing
// pending pool.
//
// It is okay to add the same transaction more than once; subsequent
// attempts will have no effect and return a nil error.
//
// It is okay to add conflicting transactions to the pool. The conflict
// will be resolved when a block lands.
func (fc *FC) AddTx(ctx context.Context, tx *bc.Tx) error {
	bcView, err := fc.store.NewViewForPrevouts(ctx, []*bc.Tx{tx})
	if err != nil {
		return errors.Wrap(err)
	}

	poolView := state.NewMemView(nil, bcView)
	poolView.Added, err = fc.store.GetPoolPrevouts(ctx, []*bc.Tx{tx})
	if err != nil {
		return errors.Wrap(err)
	}

	// Check if the transaction already exists in the blockchain.
	poolTxs, bcTxs, err := fc.store.GetTxs(ctx, tx.Hash)
	if _, ok := poolTxs[tx.Hash]; ok {
		return nil
	}
	if _, ok := bcTxs[tx.Hash]; ok {
		return nil
	}
	if err != nil {
		return errors.Wrap(err)
	}

	mv := state.NewMemView(nil, poolView)
	err = validation.ValidateTx(ctx, mv, tx, uint64(time.Now().Unix()))
	if err != nil {
		return errors.Wrap(err, "tx rejected")
	}

	err = validation.ApplyTx(ctx, mv, tx)
	if err != nil {
		return errors.Wrap(err, "applying tx")
	}

	// Update persistent tx pool state
	err = fc.applyTx(ctx, tx, mv)
	if err != nil {
		return errors.Wrap(err, "apply TX")
	}

	for _, cb := range fc.txCallbacks {
		cb(ctx, tx)
	}
	return nil
}

// applyTx updates the output set to reflect
// the effects of tx. It deletes consumed utxos
// and inserts newly-created outputs.
// Must be called inside a transaction.
func (fc *FC) applyTx(ctx context.Context, tx *bc.Tx, view *state.MemView) (err error) {
	defer metrics.RecordElapsed(time.Now())

	err = fc.store.ApplyTx(ctx, tx, view.Assets)
	return errors.Wrap(err, "applying tx to store")
}
