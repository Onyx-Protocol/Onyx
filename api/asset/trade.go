package asset

import (
	"bytes"
	"time"

	"golang.org/x/net/context"

	"chain/api/utxodb"
	"chain/errors"
	"chain/fedchain-sandbox/wire"
)

// Trade builds or adds on to a transaction for trading.
// Initially, inputs are left unconsumed, and outputs unsatisfied.
// Trading partners then satisfy and consume inputs and outputs.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Trade(ctx context.Context, prev *Tx, inputs []utxodb.Input, outputs []Output) (*Tx, error) {
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
	complete := &Tx{BlockChain: txs[0].BlockChain}
	completeWire := wire.NewMsgTx()

	for i, tx := range txs {
		if tx.BlockChain != complete.BlockChain {
			return nil, errors.New("all txs must be the same BlockChain")
		}

		complete.Inputs = append(complete.Inputs, tx.Inputs...)

		wireTx := wire.NewMsgTx()
		err := wireTx.Deserialize(bytes.NewReader(tx.Unsigned))
		if err != nil {
			return nil, errors.Wrapf(err, "deserializing tx %d", i)
		}

		for _, txin := range wireTx.TxIn {
			completeWire.TxIn = append(completeWire.TxIn, txin)
		}
		for _, txout := range wireTx.TxOut {
			completeWire.TxOut = append(completeWire.TxOut, txout)
		}
	}
	var buf bytes.Buffer
	completeWire.Serialize(&buf)
	complete.Unsigned = buf.Bytes()
	return complete, nil
}
