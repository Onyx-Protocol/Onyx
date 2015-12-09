package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/utxodb"
	"chain/errors"
	"chain/fedchain/bc"
)

// Trade builds or adds on to a transaction for trading.
// Initially, inputs are left unconsumed, and outputs unsatisfied.
// Trading partners then satisfy and consume inputs and outputs.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Trade(ctx context.Context, prev *Tx, inputs []utxodb.Input, outputs []*Output) (*Tx, error) {
	tpl, err := build(ctx, inputs, outputs, time.Hour*24)
	if err != nil {
		return nil, err
	}
	if prev != nil {
		return combine(prev, tpl)
	}
	return tpl, nil
}

func combine(txs ...*Tx) (*Tx, error) {
	if len(txs) == 0 {
		return nil, errors.New("must pass at least one tx")
	}
	completeWire := &bc.TxData{Version: bc.CurrentTransactionVersion}
	complete := &Tx{BlockChain: txs[0].BlockChain, Unsigned: completeWire}

	for _, tx := range txs {
		if tx.BlockChain != complete.BlockChain {
			return nil, errors.New("all txs must be the same BlockChain")
		}

		complete.Inputs = append(complete.Inputs, tx.Inputs...)
		complete.OutRecvs = append(complete.OutRecvs, tx.OutRecvs...)

		for _, txin := range tx.Unsigned.Inputs {
			completeWire.Inputs = append(completeWire.Inputs, txin)
		}
		for _, txout := range tx.Unsigned.Outputs {
			completeWire.Outputs = append(completeWire.Outputs, txout)
		}
	}
	return complete, nil
}
