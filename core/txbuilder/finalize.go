package txbuilder

import (
	"context"
	"time"

	"chain/core/rpcclient"
	"chain/cos"
	"chain/cos/bc"
	"chain/cos/validation"
	"chain/errors"
	chainlog "chain/log"
	"chain/metrics"
)

var (
	// ErrBadTxTemplate is returned by FinalizeTx
	ErrBadTxTemplate = errors.New("bad transaction template")

	// ErrRejected means the network rejected a tx (as a double-spend)
	ErrRejected = errors.New("transaction rejected")
)

var Generator *string

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, fc *cos.FC, txTemplate *Template) (*bc.Tx, error) {
	defer metrics.RecordElapsed(time.Now())

	if txTemplate.Unsigned == nil {
		return nil, errors.WithDetail(ErrBadTxTemplate, "missing unsigned tx")
	}

	if len(txTemplate.Inputs) > len(txTemplate.Unsigned.Inputs) {
		return nil, errors.WithDetail(ErrBadTxTemplate, "too many inputs in template")
	}

	msg, err := AssembleSignatures(txTemplate)
	if err != nil {
		return nil, errors.WithDetail(ErrBadTxTemplate, err.Error())
	}

	err = publishTx(ctx, fc, msg)
	if err != nil {
		rawtx, err2 := msg.MarshalText()
		if err2 != nil {
			// ignore marshalling errors (they should never happen anyway)
			return nil, err
		}
		return nil, errors.Wrapf(err, "tx=%s", rawtx)
	}

	return msg, nil
}

// FinalizeTxWait calls FinalizeTx and then waits for confirmation of
// the transaction.  A nil error return means the transaction is
// confirmed on the blockchain.  ErrRejected means a conflicting tx is
// on the blockchain.  context.DeadlineExceeded means ctx is an
// expiring context that timed out.
func FinalizeTxWait(ctx context.Context, fc *cos.FC, txTemplate *Template) (*bc.Tx, error) {
	// Avoid a race condition.  Calling fc.Height() here ensures that
	// when we start waiting for blocks below, we don't begin waiting at
	// block N+1 when the tx we want is in block N.
	height := fc.Height()

	tx, err := FinalizeTx(ctx, fc, txTemplate)
	if err != nil {
		return nil, err
	}

	for {
		height++
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case err := <-waitBlock(ctx, fc, height):
			if err != nil {
				// This should be impossible, since the only error produced by
				// WaitForBlock is ErrTheDistantFuture, and height is known
				// not to be in "the distant future."
				return nil, errors.Wrapf(err, "waiting for block %d", height)
			}
			// TODO(bobg): This technique is not future-proof.  The database
			// won't necessarily contain all the txs we might care about.
			// An alternative approach will be to scan through each block as
			// it lands, looking for the tx or a tx that conflicts with it.
			// For now, though, this is probably faster and simpler.
			bcTxs, err := fc.ConfirmedTxs(ctx, tx.Hash)
			if err != nil {
				return nil, errors.Wrap(err, "getting bc txs")
			}
			if _, ok := bcTxs[tx.Hash]; ok {
				// confirmed
				return tx, nil
			}

			poolTxs, err := fc.PendingTxs(ctx, tx.Hash)
			if err != nil {
				return nil, errors.Wrap(err, "getting pool txs")
			}
			if _, ok := poolTxs[tx.Hash]; !ok {
				// rejected
				return nil, ErrRejected
			}

			// still in the pool; iterate
		}
	}
}

func waitBlock(ctx context.Context, fc *cos.FC, height uint64) <-chan error {
	c := make(chan error, 1)
	go func() { c <- fc.WaitForBlock(ctx, height) }()
	return c
}

func publishTx(ctx context.Context, fc *cos.FC, msg *bc.Tx) error {
	// Make sure there is atleast one block in case client is
	// trying to finalize a tx before the genesis block has landed
	fc.WaitForBlock(ctx, 1)
	err := fc.AddTx(ctx, msg)
	if errors.Root(err) == validation.ErrBadTx {
		detail := errors.Detail(err)
		err = errors.Wrap(ErrBadTxTemplate, err)
		return errors.WithDetail(err, detail)
	} else if err != nil {
		return errors.Wrap(err, "add tx to blockchain")
	}

	if Generator != nil && *Generator != "" {
		err = rpcclient.Submit(ctx, msg)
		if err != nil {
			err = errors.Wrap(err, "generator transaction notice")
			chainlog.Error(ctx, err)

			// Return an error so that the client knows that it needs to
			// retry the request.
			return err
		}
	}
	return nil
}
