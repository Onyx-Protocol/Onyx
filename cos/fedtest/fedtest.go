package fedtest

import (
	"testing"
	"time"

	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/cos/state"
	"chain/testutil"
)

type TestDest struct {
	PrivKey                *hdkey.XKey
	PKScript, RedeemScript []byte
}

func Dest(t testing.TB) *TestDest {
	var priv *hdkey.XKey
	_, priv, err := hdkey.New()
	if err != nil {
		testutil.FatalErr(t, err)
	}

	pk, redeem, err := hdkey.Scripts([]*hdkey.XKey{priv}, nil, 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return &TestDest{
		PrivKey:      priv,
		PKScript:     pk,
		RedeemScript: redeem,
	}
}

func (d *TestDest) Sign(t testing.TB, tx *bc.TxData, index int) {
	hash := tx.HashForSig(index, bc.SigHashAll)

	ecPriv, err := d.PrivKey.ECPrivKey()
	if err != nil {
		testutil.FatalErr(t, err)
	}

	sig, err := ecPriv.Sign(hash[:])
	if err != nil {
		testutil.FatalErr(t, err)
	}
	der := append(sig.Serialize(), byte(bc.SigHashAll))
	tx.Inputs[index].InputWitness = [][]byte{der, d.RedeemScript}
}

type TestAsset struct {
	bc.AssetID
	TestDest
}

func Asset(t testing.TB) *TestAsset {
	dest := Dest(t)
	assetID := bc.ComputeAssetID(dest.PKScript, bc.Hash{})

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
			bc.NewIssuanceInput(time.Now(), time.Now().Add(time.Hour), bc.Hash{}, amount, asset.PKScript, nil, nil, nil),
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
