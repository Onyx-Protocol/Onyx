package protocol

import (
	"context"
	"fmt"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/state"
	"chain/protocol/vm"
	"chain/protocol/vm/vmutil"
	"chain/testutil"
)

func TestBadMaxIssuanceWindow(t *testing.T) {
	ctx := context.Background()
	c, b1 := newTestChain(t, time.Now())
	c.MaxIssuanceWindow = time.Second

	issueTx, _, _ := issue(t, nil, nil, 1)

	got, _, err := c.GenerateBlock(ctx, b1, state.Empty(), time.Now(), []*legacy.Tx{issueTx})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Transactions) != 0 {
		t.Error("expected issuance past max issuance window to be rejected")
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

func (d *testDest) sign(t testing.TB, tx *legacy.Tx, index uint32) {
	txsighash := tx.SigHash(index)
	prog, _ := vm.Assemble(fmt.Sprintf("0x%x TXSIGHASH EQUAL", txsighash.Bytes()))
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
	var initialBlockID bc.Hash
	assetID := bc.ComputeAssetID(cp, &initialBlockID, 1, &bc.EmptyStringHash)

	return &testAsset{
		AssetID:  assetID,
		testDest: *dest,
	}
}

func issue(t testing.TB, asset *testAsset, dest *testDest, amount uint64) (*legacy.Tx, *testAsset, *testDest) {
	if asset == nil {
		asset = newAsset(t)
	}
	if dest == nil {
		dest = newDest(t)
	}
	assetCP, _ := asset.controlProgram()
	destCP, _ := dest.controlProgram()
	tx := legacy.NewTx(legacy.TxData{
		Version: 1,
		Inputs: []*legacy.TxInput{
			legacy.NewIssuanceInput([]byte{1}, amount, nil, bc.Hash{}, assetCP, nil, nil),
		},
		Outputs: []*legacy.TxOutput{
			legacy.NewTxOutput(asset.AssetID, amount, destCP, nil),
		},
		MinTime: bc.Millis(time.Now()),
		MaxTime: bc.Millis(time.Now().Add(time.Hour)),
	})
	asset.sign(t, tx, 0)

	return tx, asset, dest
}
