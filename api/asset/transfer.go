package asset

import (
	"bytes"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/utxodb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain-sandbox/wire"
	"chain/metrics"
)

// All UTXOs in the system.
var utxoDB = utxodb.New(sqlUTXODB{})

// errors returned by Transfer
var (
	ErrBadOutDest       = errors.New("invalid output destinations")
	ErrTransferMismatch = errors.New("input values don't match output values")
	ErrBadTxHex         = errors.New("invalid raw transaction hex")
)

// Transfer creates a transaction that
// transfers assets from input buckets
// to output buckets or addresses.
func Transfer(ctx context.Context, inputs []utxodb.Input, outputs []*Output) (*Tx, error) {
	defer metrics.RecordElapsed(time.Now())
	if err := checkTransferParity(inputs, outputs); err != nil {
		return nil, err
	}
	return build(ctx, inputs, outputs, time.Minute)
}

func build(ctx context.Context, inputs []utxodb.Input, outs []*Output, ttl time.Duration) (*Tx, error) {
	if err := validateOutputs(outs); err != nil {
		return nil, err
	}

	tx := wire.NewMsgTx()

	reserved, change, err := utxoDB.Reserve(ctx, inputs, ttl)
	if err != nil {
		return nil, errors.Wrap(err, "reserve")
	}

	for _, c := range change {
		outs = append(outs, &Output{
			BucketID: c.Input.BucketID,
			AssetID:  c.Input.AssetID,
			Amount:   int64(c.Amount),
			isChange: true,
		})
	}

	for _, utxo := range reserved {
		tx.AddTxIn(wire.NewTxIn(&utxo.Outpoint, []byte{}))
	}

	var outRecvs []*utxodb.Receiver
	for i, out := range outs {
		asset, err := wire.NewHash20FromStr(out.AssetID)
		if err != nil {
			return nil, errors.WithDetailf(appdb.ErrBadAsset, "asset id: %v", out.AssetID)
		}

		pkScript, receiver, err := out.PKScript(ctx)
		if err != nil {
			return nil, errors.WithDetailf(err, "output %d", i)
		}

		tx.AddTxOut(wire.NewTxOut(asset, out.Amount, pkScript))
		outRecvs = append(outRecvs, receiver)
	}

	var buf bytes.Buffer
	tx.Serialize(&buf)

	txInputs, err := makeTransferInputs(ctx, tx, reserved)
	if err != nil {
		return nil, err
	}

	appTx := &Tx{
		Unsigned:   buf.Bytes(),
		BlockChain: "sandbox",
		Inputs:     txInputs,
		OutRecvs:   outRecvs,
	}

	return appTx, nil
}

func validateOutputs(outputs []*Output) error {
	for i, out := range outputs {
		if (out.BucketID == "") == (out.Address == "") {
			return errors.WithDetailf(ErrBadOutDest, "output index=%d", i)
		}
	}
	return nil
}

func checkTransferParity(ins []utxodb.Input, outs []*Output) error {
	parity := make(map[string]int64)
	for _, in := range ins {
		parity[in.AssetID] += int64(in.Amount)
	}
	for _, out := range outs {
		parity[out.AssetID] -= int64(out.Amount)
	}

	for asset, amt := range parity {
		if amt != 0 {
			return errors.WithDetailf(ErrBadTx, "asset %q does not balance", asset)
		}
	}

	return nil
}

// makeTransferInputs creates the array of inputs
// that contain signatures along with the
// data needed to generate them
func makeTransferInputs(ctx context.Context, tx *wire.MsgTx, utxos []*utxodb.UTXO) ([]*Input, error) {
	defer metrics.RecordElapsed(time.Now())
	var inputs []*Input
	for _, utxo := range utxos {
		input, err := addressInput(ctx, utxo, tx)
		if err != nil {
			return nil, errors.Wrap(err, "compute input")
		}
		inputs = append(inputs, input)
	}
	return inputs, nil
}

func addressInput(ctx context.Context, u *utxodb.UTXO, tx *wire.MsgTx) (*Input, error) {
	var buf bytes.Buffer
	tx.Serialize(&buf)

	addrInfo, err := appdb.AddrInfo(ctx, u.BucketID)
	if err != nil {
		return nil, errors.Wrap(err, "get addr info")
	}

	// TODO(kr): for key rotation, pull keys out of utxo
	// instead of the bucket (addrInfo).
	signers := hdkey.Derive(addrInfo.Keys, appdb.ReceiverPath(addrInfo, u.AddrIndex[:]))
	redeemScript, err := hdkey.RedeemScript(signers, addrInfo.SigsRequired)
	if err != nil {
		return nil, errors.Wrap(err, "compute redeem script")
	}

	in := &Input{
		WalletID:      addrInfo.WalletID,
		RedeemScript:  redeemScript,
		SignatureData: wire.DoubleSha256(buf.Bytes()),
		Sigs:          inputSigs(signers),
	}
	return in, nil
}

// CancelReservations cancels any existing reservations
// for the given outpoints.
func CancelReservations(ctx context.Context, outpoints []wire.OutPoint) {
	utxoDB.Cancel(ctx, outpoints)
}
