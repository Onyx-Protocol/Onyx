package protocol

import (
	"context"
	"testing"
	"time"

	"chain/crypto/ed25519"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/vmutil"
	"chain/testutil"
)

func TestIdempotentAddTx(t *testing.T) {
	ctx := context.Background()
	c, b1 := newTestChain(t, time.Now())

	issueTx, _, _ := issue(t, nil, nil, 1)

	err := c.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// still idempotent after block lands
	block, tree, err := c.GenerateBlock(ctx, b1, state.Empty(), time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = c.CommitBlock(ctx, block, tree)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = c.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func TestAddTx(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestChain(t, time.Now())

	issueTx, _, dest1 := issue(t, nil, nil, 1)
	err := c.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	transferTx := transfer(t, stateOut(issueTx, 0), dest1, newDest(t))

	err = c.AddTx(ctx, transferTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

type testDest struct {
	privKey                ed25519.PrivateKey
	pkScript, redeemScript []byte
}

func newDest(t testing.TB) *testDest {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	pkScript, redeem, err := vmutil.TxScripts([]ed25519.PublicKey{pub}, 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return &testDest{
		privKey:      priv,
		pkScript:     pkScript,
		redeemScript: redeem,
	}
}

func (d *testDest) sign(t testing.TB, tx *bc.TxData, index int) {
	hash := tx.HashForSig(index, bc.SigHashAll)
	sig := ed25519.Sign(d.privKey, hash[:])
	tx.Inputs[index].InputWitness = [][]byte{sig, d.redeemScript}
}

type testAsset struct {
	bc.AssetID
	testDest
}

func newAsset(t testing.TB) *testAsset {
	dest := newDest(t)
	assetID := bc.ComputeAssetID(dest.pkScript, bc.Hash{}, 1)

	return &testAsset{
		AssetID:  assetID,
		testDest: *dest,
	}
}

func issue(t testing.TB, asset *testAsset, dest *testDest, amount uint64) (*bc.Tx, *testAsset, *testDest) {
	if asset == nil {
		asset = newAsset(t)
	}
	if dest == nil {
		dest = newDest(t)
	}
	tx := &bc.TxData{
		Version: bc.CurrentTransactionVersion,
		Inputs: []*bc.TxInput{
			bc.NewIssuanceInput(time.Now(), time.Now().Add(time.Hour), bc.Hash{}, amount, asset.pkScript, nil, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(asset.AssetID, amount, dest.pkScript, nil),
		},
	}
	asset.sign(t, tx, 0)

	return bc.NewTx(*tx), asset, dest
}

func transfer(t testing.TB, out *state.Output, from, to *testDest) *bc.Tx {
	tx := &bc.TxData{
		Version: bc.CurrentTransactionVersion,
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(out.Hash, out.Index, nil, out.AssetID, out.Amount, out.ControlProgram, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(out.AssetID, out.Amount, to.pkScript, nil),
		},
	}
	from.sign(t, tx, 0)

	return bc.NewTx(*tx)
}

func stateOut(tx *bc.Tx, index int) *state.Output {
	return &state.Output{
		TxOutput: *tx.Outputs[index],
		Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(index)},
	}
}
