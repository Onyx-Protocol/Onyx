package state

import (
	"reflect"
	"testing"
	"time"

	"chain/protocol/bc"
	"chain/protocol/bc/bctest"
	"chain/protocol/bc/legacy"
)

func TestApplyTxSpend(t *testing.T) {
	assetID := bc.AssetID{}
	sourceID := bc.NewHash([32]byte{0x01, 0x02, 0x03})
	sc := legacy.SpendCommitment{
		AssetAmount:    bc.AssetAmount{AssetId: &assetID, Amount: 100},
		SourceID:       sourceID,
		SourcePosition: 0,
		VMVersion:      1,
		ControlProgram: nil,
		RefDataHash:    bc.Hash{},
	}
	spentOutputID, err := legacy.ComputeOutputID(&sc)
	if err != nil {
		t.Fatal(err)
	}

	snap := Empty()
	snap.Tree.Insert(spentOutputID.Bytes())

	tx := legacy.MapTx(&legacy.TxData{
		Version: 1,
		Inputs: []*legacy.TxInput{
			legacy.NewSpendInput(nil, sourceID, assetID, 100, 0, nil, bc.Hash{}, nil),
		},
		Outputs: []*legacy.TxOutput{},
	})

	// Apply the spend transaction.
	err = snap.ApplyTx(tx)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Tree.Contains(spentOutputID.Bytes()) {
		t.Error("snapshot contains spent prevout")
	}
	err = snap.ApplyTx(tx)
	if err == nil {
		t.Error("expected error applying spend twice, got nil")
	}
}

func TestApplyIssuanceTwice(t *testing.T) {
	snap := Empty()
	issuance := legacy.MapTx(&bctest.NewIssuanceTx(t, bc.EmptyStringHash).TxData)
	err := snap.ApplyTx(issuance)
	if err != nil {
		t.Fatal(err)
	}
	err = snap.ApplyTx(issuance)
	if err == nil {
		t.Errorf("expected error for duplicate nonce, got %s", err)
	}
}

func TestCopySnapshot(t *testing.T) {
	snap := Empty()
	err := snap.ApplyTx(legacy.MapTx(&bctest.NewIssuanceTx(t, bc.EmptyStringHash).TxData))
	if err != nil {
		t.Fatal(err)
	}
	dupe := Copy(snap)
	if !reflect.DeepEqual(dupe, snap) {
		t.Errorf("got %#v, want %#v", dupe, snap)
	}
}

func TestApplyBlock(t *testing.T) {
	// Setup a snapshot with a nonce with a known expiry.
	maxTime := bc.Millis(time.Now().Add(5 * time.Minute))
	issuance := bctest.NewIssuanceTx(t, bc.EmptyStringHash, func(tx *legacy.Tx) {
		tx.MaxTime = maxTime
	})
	snap := Empty()
	err := snap.ApplyTx(legacy.MapTx(&issuance.TxData))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(snap.Nonces); n != 1 {
		t.Errorf("got %d nonces, want 1", n)
	}

	// Land a block later than the issuance's max time.
	block := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			TimestampMS: maxTime + 1,
		},
	}
	err = snap.ApplyBlock(legacy.MapBlock(block))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(snap.Nonces); n != 0 {
		t.Errorf("got %d nonces, want 0", n)
	}
}
