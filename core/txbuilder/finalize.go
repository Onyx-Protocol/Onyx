package txbuilder

import (
	"context"

	"chain/errors"
	chainlog "chain/log"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/validation"
)

var (
	// ErrRejected means the network rejected a tx (as a double-spend)
	ErrRejected = errors.New("transaction rejected")

	ErrMissingRawTx        = errors.New("missing raw tx")
	ErrBadInstructionCount = errors.New("too many signing instructions in template")
)

var Generator *rpc.Client

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, c *protocol.Chain, tx *bc.Tx) error {
	err := publishTx(ctx, c, tx)
	if err != nil {
		rawtx, err2 := tx.MarshalText()
		if err2 != nil {
			// ignore marshalling errors (they should never happen anyway)
			return err
		}
		return errors.Wrapf(err, "tx=%s", rawtx)
	}

	return nil
}

func publishTx(ctx context.Context, c *protocol.Chain, msg *bc.Tx) error {
	// Make sure there is atleast one block in case client is
	// trying to finalize a tx before the initial block has landed
	c.WaitForBlock(1)

	var err error
	if Generator != nil {
		// If this transaction is valid, ValidateTxCached will store it in the cache.
		err := c.ValidateTxCached(msg)
		if err != nil {
			return errors.Wrap(err, "tx rejected")
		}

		err = Generator.Call(ctx, "/rpc/submit", msg, nil)
		if err != nil {
			err = errors.Wrap(err, "generator transaction notice")
			chainlog.Error(ctx, err)

			// Return an error so that the client knows that it needs to
			// retry the request.
			return err
		}
	} else {
		err = c.AddTx(ctx, msg)
		if errors.Root(err) == validation.ErrBadTx {
			detail := errors.Detail(err)
			err = errors.Wrap(ErrRejected, err)
			return errors.WithDetail(err, detail)
		} else if err != nil {
			return errors.Wrap(err, "add tx to blockchain")
		}
	}
	return nil
}
