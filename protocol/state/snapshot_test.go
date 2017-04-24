package state

import (
	"testing"

	"chain/protocol/bc"
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

	tx, err := legacy.MapTx(&legacy.TxData{
		Version: 1,
		Inputs: []*legacy.TxInput{
			legacy.NewSpendInput(nil, sourceID, assetID, 100, 0, nil, bc.Hash{}, nil),
		},
		Outputs: []*legacy.TxOutput{},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Apply the spend transaction.
	err = snap.ApplyTx(tx)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Tree.Contains(spentOutputID.Bytes()) {
		t.Error("snapshot contains spent prevout")
	}
}
