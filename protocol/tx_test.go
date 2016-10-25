package protocol

import (
	"context"
	"fmt"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/validation"
	"chain/protocol/vm"
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
	block, tree, err := c.GenerateBlock(ctx, b1, state.NewSnapshot(b1.Hash()), time.Now())
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

func TestAddTxBadMaxIssuanceWindow(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestChain(t, time.Now())
	c.MaxIssuanceWindow = time.Second

	issueTx, _, _ := issue(t, nil, nil, 1)
	err := c.AddTx(ctx, issueTx)
	if errors.Root(err) != validation.ErrBadTx {
		t.Errorf("expected err to have Root %s, got %s", validation.ErrBadTx, errors.Root(err))
	}
}

type testDest struct {
	privKey ed25519.PrivateKey
}

func newDest(t testing.TB) *testDest {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return &testDest{
		privKey: priv,
	}
}

func (d *testDest) sign(t testing.TB, tx *bc.TxData, index int) {
	txsighash := tx.HashForSig(index)
	prog, _ := vm.Assemble(fmt.Sprintf("0x%x TXSIGHASH EQUAL", txsighash[:]))
	h := sha3.Sum256(prog)
	sig := ed25519.Sign(d.privKey, h[:])
	tx.Inputs[index].SetArguments([][]byte{vm.Int64Bytes(0), sig, prog})
}

func (d testDest) controlProgram() ([]byte, error) {
	pub := d.privKey.Public().(ed25519.PublicKey)
	return vmutil.P2SPMultiSigProgram([]ed25519.PublicKey{pub}, 1)
}

type testAsset struct {
	bc.AssetID
	testDest
}

func newAsset(t testing.TB) *testAsset {
	dest := newDest(t)
	cp, _ := dest.controlProgram()
	assetID := bc.ComputeAssetID(cp, bc.Hash{}, 1)

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
	assetCP, _ := asset.controlProgram()
	destCP, _ := dest.controlProgram()
	tx := &bc.TxData{
		Version: bc.CurrentTransactionVersion,
		Inputs: []*bc.TxInput{
			bc.NewIssuanceInput([]byte{1}, amount, nil, bc.Hash{}, assetCP, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(asset.AssetID, amount, destCP, nil),
		},
		MinTime: bc.Millis(time.Now()),
		MaxTime: bc.Millis(time.Now().Add(time.Hour)),
	}
	asset.sign(t, tx, 0)

	return bc.NewTx(*tx), asset, dest
}

func transfer(t testing.TB, out *state.Output, from, to *testDest) *bc.Tx {
	cp, _ := to.controlProgram()
	tx := &bc.TxData{
		Version: bc.CurrentTransactionVersion,
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(out.Hash, out.Index, nil, out.AssetID, out.Amount, out.ControlProgram, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(out.AssetID, out.Amount, cp, nil),
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
