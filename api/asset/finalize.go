package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/rpcclient"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	chainlog "chain/log"
	"chain/metrics"
)

// ErrBadTx is returned by FinalizeTx
var ErrBadTx = errors.New("bad transaction template")

var Generator *string

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, txTemplate *txbuilder.Template) (*bc.Tx, error) {
	defer metrics.RecordElapsed(time.Now())

	if len(txTemplate.Inputs) > len(txTemplate.Unsigned.Inputs) {
		return nil, errors.WithDetail(ErrBadTx, "too many inputs in template")
	}

	msg, err := txbuilder.AssembleSignatures(txTemplate)
	if err != nil {
		return nil, errors.WithDetail(ErrBadTx, err.Error())
	}

	err = publishTx(ctx, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func publishTx(ctx context.Context, msg *bc.Tx) error {
	err := fc.AddTx(ctx, msg)
	if err != nil {
		return errors.Wrap(err, "add tx to fedchain")
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

func addAccountData(ctx context.Context, tx *bc.Tx) error {
	txdbMap := make(map[bc.Outpoint]*txdb.Output, len(tx.Outputs))
	txdbOuts := make([]*txdb.Output, 0, len(tx.Outputs))
	for i, out := range tx.Outputs {
		txdbOutput := &txdb.Output{
			Output: state.Output{
				TxOutput: *out,
				Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(i)},
			},
		}
		txdbMap[txdbOutput.Outpoint] = txdbOutput
		txdbOuts = append(txdbOuts, txdbOutput)
	}

	err := loadAccountInfoFromAddrs(ctx, txdbMap)
	if err != nil {
		return errors.Wrap(err, "loading account info from addresses")
	}

	err = txdb.InsertAccountOutputs(ctx, txdbOuts)
	return errors.Wrap(err, "updating pool outputs")
}
