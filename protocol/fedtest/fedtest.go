package fedtest

import (
	"testing"
	"time"

	"chain/crypto/ed25519"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/vmutil"
	"chain/testutil"
)

type TestDest struct {
	PrivKey                ed25519.PrivateKey
	PKScript, RedeemScript []byte
}

func Dest(t testing.TB) *TestDest {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	pkScript, redeem, err := vmutil.TxScripts([]ed25519.PublicKey{pub}, 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return &TestDest{
		PrivKey:      priv,
		PKScript:     pkScript,
		RedeemScript: redeem,
	}
}

func (d *TestDest) Sign(t testing.TB, tx *bc.TxData, index int) {
	hash := tx.HashForSig(index, bc.SigHashAll)
	sig := ed25519.Sign(d.PrivKey, hash[:])
	tx.Inputs[index].InputWitness = [][]byte{sig, d.RedeemScript}
}

type TestAsset struct {
	bc.AssetID
	TestDest
}

func Asset(t testing.TB) *TestAsset {
	dest := Dest(t)
	assetID := bc.ComputeAssetID(dest.PKScript, bc.Hash{}, 1)

	return &TestAsset{
		AssetID:  assetID,
		TestDest: *dest,
	}
}

func Issue(t testing.TB, asset *TestAsset, dest *TestDest, amount uint64) (*bc.Tx, *TestAsset, *TestDest) {
	if asset == nil {
		asset = Asset(t)
	}
	if dest == nil {
		dest = Dest(t)
	}
	tx := &bc.TxData{
		Version: bc.CurrentTransactionVersion,
		Inputs: []*bc.TxInput{
			bc.NewIssuanceInput(time.Now(), time.Now().Add(time.Hour), bc.Hash{}, amount, asset.PKScript, nil, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(asset.AssetID, amount, dest.PKScript, nil),
		},
	}
	asset.Sign(t, tx, 0)

	return bc.NewTx(*tx), asset, dest
}

func Transfer(t testing.TB, out *state.Output, from, to *TestDest) *bc.Tx {
	tx := &bc.TxData{
		Version: bc.CurrentTransactionVersion,
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(out.Hash, out.Index, nil, out.AssetID, out.Amount, out.ControlProgram, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(out.AssetID, out.Amount, to.PKScript, nil),
		},
	}
	from.Sign(t, tx, 0)

	return bc.NewTx(*tx)
}

func StateOut(tx *bc.Tx, index int) *state.Output {
	return &state.Output{
		TxOutput: *tx.Outputs[index],
		Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(index)},
	}
}
