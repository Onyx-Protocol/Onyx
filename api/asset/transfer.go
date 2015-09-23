package asset

import (
	"bytes"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain-sandbox/wire"
	"chain/metrics"
)

// errors returned by Transfer
var (
	ErrBadOutDest       = errors.New("invalid output destinations")
	ErrTransferMismatch = errors.New("input values don't match output values")
)

// TransferInput is a user input struct used in Transfer.
type TransferInput struct {
	AssetID  string `json:"asset_id"`
	BucketID string `json:"bucket_id"`
	TxID     string `json:"transaction_id"`
	Amount   int64
}

// Transfer creates a transaction that
// transfers assets from input buckets
// to output buckets or addresses.
func Transfer(ctx context.Context, inputs []TransferInput, outputs []Output) (*Tx, error) {
	defer metrics.RecordElapsed(time.Now())
	if err := validateTransfer(inputs, outputs); err != nil {
		return nil, err
	}

	var (
		outs     = outputs
		tx       = wire.NewMsgTx()
		allUTXOs []*appdb.UTXO
	)

	for _, in := range inputs {
		var (
			utxos []*appdb.UTXO
			sum   int64
			err   error
		)
		if in.TxID != "" {
			utxos, sum, err = appdb.ReserveTxUTXOs(ctx, in.AssetID, in.BucketID, in.TxID, in.Amount, time.Minute)
		} else {
			utxos, sum, err = appdb.ReserveUTXOs(ctx, in.AssetID, in.BucketID, in.Amount, time.Minute)
		}
		if err != nil {
			err = errors.WithDetailf(err, "bucket=%v asset=%v amount=%v txid=%v",
				in.AssetID, in.BucketID, in.Amount, in.TxID)
			return nil, err
		}

		allUTXOs = append(allUTXOs, utxos...)

		for _, utxo := range utxos {
			tx.AddTxIn(wire.NewTxIn(utxo.OutPoint, []byte{}))
		}

		if sum > in.Amount {
			outs = append(outs, Output{
				BucketID: in.BucketID,
				AssetID:  in.AssetID,
				Amount:   sum - in.Amount,
				isChange: true,
			})
		}
	}

	for i, out := range outs {
		asset, err := wire.NewHash20FromStr(out.AssetID)
		if err != nil {
			return nil, errors.WithDetailf(appdb.ErrBadAsset, "asset id: %v", out.AssetID)
		}

		pkScript, err := out.PkScript(ctx)
		if err != nil {
			return nil, errors.WithDetailf(err, "output %d: %v", i, err.Error())
		}

		tx.AddTxOut(wire.NewTxOut(asset, out.Amount, pkScript))
	}

	var buf bytes.Buffer
	tx.Serialize(&buf)

	txInputs, err := makeTransferInputs(ctx, tx, allUTXOs)
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

func validateTransfer(inputs []TransferInput, outputs []Output) error {
	parity := make(map[string]int64)
	for _, in := range inputs {
		parity[in.AssetID] -= in.Amount
	}
	for i, out := range outputs {
		if (out.BucketID == "") == (out.Address == "") {
			return errors.WithDetailf(ErrBadOutDest, "output index=%d", i)
		}
		parity[out.AssetID] += out.Amount
	}
	for _, amt := range parity {
		if amt != 0 {
			return ErrTransferMismatch
		}
	}
	return nil
}

// makeTransferInputs creates the array of inputs
// that contain signatures along with the
// data needed to generate them
func makeTransferInputs(ctx context.Context, tx *wire.MsgTx, utxos []*appdb.UTXO) ([]*Input, error) {
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
