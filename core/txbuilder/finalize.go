package txbuilder

import (
	"context"
	"time"

	"chain/core/rpcclient"
	"chain/errors"
	chainlog "chain/log"
	"chain/metrics"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/validation"
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
func FinalizeTx(ctx context.Context, c *protocol.Chain, txTemplate *Template) (*bc.Tx, error) {
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

	err = publishTx(ctx, c, msg)
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

func publishTx(ctx context.Context, c *protocol.Chain, msg *bc.Tx) error {
	// Make sure there is atleast one block in case client is
	// trying to finalize a tx before the genesis block has landed
	c.WaitForBlock(ctx, 1)
	err := c.AddTx(ctx, msg)
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
