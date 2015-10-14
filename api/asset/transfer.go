package asset

import (
	"bytes"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/utxodb"
	"chain/errors"
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
func Transfer(ctx context.Context, inputs []utxodb.Input, outputs []Output) (*Tx, error) {
	defer metrics.RecordElapsed(time.Now())
	if err := checkTransferParity(inputs, outputs); err != nil {
		return nil, err
	}
	return build(ctx, inputs, outputs, time.Minute)
}

func build(ctx context.Context, inputs []utxodb.Input, outs []Output, ttl time.Duration) (*Tx, error) {
	if err := validateOutputs(outs); err != nil {
		return nil, err
	}

	tx := wire.NewMsgTx()

	reserved, change, err := utxoDB.Reserve(ctx, inputs, ttl)
	if err != nil {
		return nil, errors.Wrap(err, "reserve")
	}

	for _, c := range change {
		outs = append(outs, Output{
			BucketID: c.Input.BucketID,
			AssetID:  c.Input.AssetID,
			Amount:   int64(c.Amount),
			isChange: true,
		})
	}

	for _, utxo := range reserved {
		tx.AddTxIn(wire.NewTxIn(&utxo.Outpoint, []byte{}))
	}

	for i, out := range outs {
		asset, err := wire.NewHash20FromStr(out.AssetID)
		if err != nil {
			return nil, errors.WithDetailf(appdb.ErrBadAsset, "asset id: %v", out.AssetID)
		}

		pkScript, err := out.PkScript(ctx)
		if err != nil {
			return nil, errors.WithDetailf(err, "output %d", i)
		}

		tx.AddTxOut(wire.NewTxOut(asset, out.Amount, pkScript))
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
	}

	return appTx, nil
}

func validateOutputs(outputs []Output) error {
	for i, out := range outputs {
		if (out.BucketID == "") == (out.Address == "") {
			return errors.WithDetailf(ErrBadOutDest, "output index=%d", i)
		}
	}
	return nil
}

func checkTransferParity(ins []utxodb.Input, outs []Output) error {
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
	var addrIDs []string
	for _, utxo := range utxos {
		addrIDs = append(addrIDs, utxo.AddressID)
	}

	addrs, err := appdb.AddressesByID(ctx, addrIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "for addresses=%+v", addrIDs)
	}

	addrMap := make(map[string]*appdb.Address)
	for _, a := range addrs {
		addrMap[a.ID] = a
	}

	var inputs []*Input
	for _, utxo := range utxos {
		addr, ok := addrMap[utxo.AddressID]
		if !ok {
			return nil, errors.New("missing address")
		}
		inputs = append(inputs, addressInput(addr, tx))
	}

	return inputs, nil
}

func addressInput(a *appdb.Address, tx *wire.MsgTx) *Input {
	var buf bytes.Buffer
	tx.Serialize(&buf)

	return &Input{
		WalletID:      a.WalletID,
		RedeemScript:  a.RedeemScript,
		SignatureData: wire.DoubleSha256(buf.Bytes()),
		Sigs:          inputSigs(Signers(a.Keys, ReceiverPath(a))),
	}
}

// CancelReservations cancels any existing reservations
// for the given outpoints.
func CancelReservations(ctx context.Context, outpoints []wire.OutPoint) {
	utxoDB.Cancel(ctx, outpoints)
}
