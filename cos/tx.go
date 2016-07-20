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
//
// It is an error to call AddTx before the genesis block has landed.
// Use WaitForBlock to guarantee this.
func (fc *FC) AddTx(ctx context.Context, tx *bc.Tx) error {
	prev, err := fc.store.LatestBlock(ctx)
	if err != nil {
		return errors.Wrap(err, "fetch latest block")
	}
	tree, err := fc.store.StateTree(ctx, prev.Height)
	if err != nil {
		return errors.Wrap(err, "loading state tree")
	}

	// tx's prevouts may consume outputs from other transactions
	// in the pool. We load any applicable pool prevouts to supplement
	// the state tree.
	poolPrevouts, err := fc.getPoolPrevouts(ctx, tx)
	if err != nil {
		return errors.Wrap(err)
	}

	// Check if the transaction already exists in the tx pool.
	poolTxs, err := fc.pool.GetTxs(ctx, tx.Hash)
	if err != nil {
		return errors.Wrap(err)
	}
	if _, ok := poolTxs[tx.Hash]; ok {
		return nil
	}
	// Check if the transaction already exists in the blockchain.
	bcTxs, err := fc.store.GetTxs(ctx, tx.Hash)
	if err != nil {
		return errors.Wrap(err)
	}
	if _, ok := bcTxs[tx.Hash]; ok {
		return nil
	}

	err = validation.ValidateTx(tree, poolPrevouts, tx, bc.NowMillis())
	if err != nil {
		return errors.Wrap(err, "tx rejected")
	}

	err = validation.ApplyTx(tree, tx)
	if err != nil {
		return errors.Wrap(err, "applying tx")
	}

	// Update persistent tx pool state
	err = fc.applyTx(ctx, tx)
	if err != nil {
		return errors.Wrap(err, "apply tx")
	}

	for _, cb := range fc.txCallbacks {
		cb(ctx, tx)
	}
	return nil
}

// applyTx updates the pool to contain the provided tx.
func (fc *FC) applyTx(ctx context.Context, tx *bc.Tx) (err error) {
	defer metrics.RecordElapsed(time.Now())

	err = fc.pool.Insert(ctx, tx)
	return errors.Wrap(err, "applying tx to store")
}

// getPoolPrevouts takes a transaction and looks up all of its prevouts
// in the pool. It returns all of the matching outpoints that it finds
// in the pool.
//
// It does not verify that the outputs are unspent in the pool.
func (fc *FC) getPoolPrevouts(ctx context.Context, tx *bc.Tx) (prevouts state.OutputSet, err error) {
	hashes := make([]bc.Hash, 0, len(tx.Inputs))
	for _, txin := range tx.Inputs {
		hashes = append(hashes, txin.Previous.Hash)
	}

	txs, err := fc.pool.GetTxs(ctx, hashes...)
	if err != nil {
		return prevouts, err
	}

	var poolPrevouts []*state.Output
	for _, txin := range tx.Inputs {
		prevTx, ok := txs[txin.Previous.Hash]
		if !ok {
			continue
		}
		if txin.Previous.Index >= uint32(len(prevTx.Outputs)) {
			continue
		}
		o := prevTx.Outputs[txin.Previous.Index]
		poolPrevouts = append(poolPrevouts, state.NewOutput(*o, txin.Previous))
	}
	return state.NewOutputSet(poolPrevouts...), nil
}
