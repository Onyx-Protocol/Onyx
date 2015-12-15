package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/utxodb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
	"chain/metrics"
)

// All UTXOs in the system.
var utxoDB = utxodb.New(sqlUTXODB{})

// errors returned by Build
var ErrBadOutDest = errors.New("invalid output destinations")

// Build builds or adds on to a transaction.
// Initially, inputs are left unconsumed, and outputs unsatisfied.
// Build partners then satisfy and consume inputs and outputs.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Build(ctx context.Context, prev *Tx, inputs []utxodb.Input, outputs []*Output, ttl time.Duration) (*Tx, error) {
	if ttl < time.Minute {
		ttl = time.Minute
	}
	tpl, err := build(ctx, inputs, outputs, ttl)
	if err != nil {
		return nil, err
	}
	if prev != nil {
		return combine(prev, tpl)
	}
	return tpl, nil
}

func build(ctx context.Context, inputs []utxodb.Input, outs []*Output, ttl time.Duration) (*Tx, error) {
	if err := validateOutputs(outs); err != nil {
		return nil, err
	}

	tx := &bc.TxData{Version: bc.CurrentTransactionVersion}

	reserved, change, err := utxoDB.Reserve(ctx, inputs, ttl)
	if err != nil {
		return nil, errors.Wrap(err, "reserve")
	}

	for _, c := range change {
		outs = append(outs, &Output{
			AccountID: c.Input.AccountID,
			AssetID:   c.Input.AssetID,
			Amount:    c.Amount,
			isChange:  true,
		})
	}

	for _, utxo := range reserved {
		tx.Inputs = append(tx.Inputs, &bc.TxInput{Previous: utxo.Outpoint})
	}

	var outRecvs []*utxodb.Receiver
	for i, out := range outs {
		hash, err := bc.ParseHash(out.AssetID)
		if err != nil {
			return nil, errors.WithDetailf(appdb.ErrBadAsset, "asset id: %v", out.AssetID)
		}
		asset := bc.AssetID(hash)

		pkScript, receiver, err := out.PKScript(ctx)
		if err != nil {
			return nil, errors.WithDetailf(err, "output %d", i)
		}

		tx.Outputs = append(tx.Outputs, &bc.TxOutput{
			AssetID:  asset,
			Value:    out.Amount,
			Script:   pkScript,
			Metadata: out.Metadata,
		})
		outRecvs = append(outRecvs, receiver)
	}

	txInputs, err := makeTransferInputs(ctx, tx, reserved)
	if err != nil {
		return nil, err
	}

	appTx := &Tx{
		Unsigned:   tx,
		BlockChain: "sandbox",
		Inputs:     txInputs,
		OutRecvs:   outRecvs,
	}

	return appTx, nil
}

func validateOutputs(outputs []*Output) error {
	for i, out := range outputs {
		if (out.AccountID == "") == (out.Address == "") {
			return errors.WithDetailf(ErrBadOutDest, "output index=%d", i)
		}
	}
	return nil
}

// makeTransferInputs creates the array of inputs
// that contain signatures along with the
// data needed to generate them
func makeTransferInputs(ctx context.Context, tx *bc.TxData, utxos []*utxodb.UTXO) ([]*Input, error) {
	defer metrics.RecordElapsed(time.Now())
	var inputs []*Input
	for i, utxo := range utxos {
		input, err := addressInput(ctx, utxo, tx, i)
		if err != nil {
			return nil, errors.Wrap(err, "compute input")
		}
		inputs = append(inputs, input)
	}
	return inputs, nil
}

func addressInput(ctx context.Context, u *utxodb.UTXO, tx *bc.TxData, idx int) (*Input, error) {
	addrInfo, err := appdb.AddrInfo(ctx, u.AccountID)
	if err != nil {
		return nil, errors.Wrap(err, "get addr info")
	}

	// TODO(kr): for key rotation, pull keys out of utxo
	// instead of the account (addrInfo).
	signers := hdkey.Derive(addrInfo.Keys, appdb.ReceiverPath(addrInfo, u.AddrIndex[:]))
	redeemScript, err := hdkey.RedeemScript(signers, addrInfo.SigsRequired)
	if err != nil {
		return nil, errors.Wrap(err, "compute redeem script")
	}

	hash, err := txscript.CalcSignatureHash(tx, idx, redeemScript, txscript.SigHashAll)
	if err != nil {
		return nil, errors.Wrap(err, "calculating signature hash")
	}

	in := &Input{
		ManagerNodeID: addrInfo.ManagerNodeID,
		RedeemScript:  redeemScript,
		SignatureData: hash,
		Sigs:          inputSigs(signers),
	}
	return in, nil
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

// CancelReservations cancels any existing reservations
// for the given outpoints.
func CancelReservations(ctx context.Context, outpoints []bc.Outpoint) {
	utxoDB.Cancel(ctx, outpoints)
}
