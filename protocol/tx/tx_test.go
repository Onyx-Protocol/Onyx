package tx

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/protocol/bc"
)

func TestMapTx(t *testing.T) {
	// sample data copied from protocol/bc/transaction_test.go

	issuanceScript := []byte{1}
	initialBlockHashHex := "03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"
	initialBlockHash := mustDecodeHash(initialBlockHashHex)

	assetID := bc.ComputeAssetID(issuanceScript, initialBlockHash, 1, bc.EmptyStringHash)

	amt0 := uint64(600000000000)
	amt1 := uint64(400000000000)

	out0 := bc.NewTxOutput(assetID, amt0, []byte{1}, nil)
	out1 := bc.NewTxOutput(assetID, amt1, []byte{2}, nil)

	oldOuts := []*bc.TxOutput{
		out0,
		out1,
	}

	oldTx := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(bc.ComputeOutputID(mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), 0), nil, bc.AssetID{}, 1000000000000, []byte{1}, []byte("input")),
		},
		Outputs:       oldOuts,
		MinTime:       1492590000,
		MaxTime:       1492590591,
		ReferenceData: []byte("distribution"),
	}

	header, entryMap, err := mapTx(oldTx)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(spew.Sdump(entryMap))

	if header.body.Version != 1 {
		t.Errorf("header.body.Version is %d, expected 1", header.body.Version)
	}
	if header.body.MinTimeMS != oldTx.MinTime {
		t.Errorf("header.body.MinTimeMS is %d, expected %d", header.body.MinTimeMS, oldTx.MinTime)
	}
	if header.body.MaxTimeMS != oldTx.MaxTime {
		t.Errorf("header.body.MaxTimeMS is %d, expected %d", header.body.MaxTimeMS, oldTx.MaxTime)
	}
	if len(header.body.Results) != len(oldOuts) {
		t.Errorf("header.body.Results contains %d item(s), expected %d", len(header.body.Results), len(oldOuts))
	}

	for i, oldOut := range oldOuts {
		if resultEntry, ok := entryMap[header.body.Results[i]]; ok {
			if newOut, ok := resultEntry.(*output); ok {
				if newOut.body.Source.Value != oldOut.AssetAmount {
					t.Errorf("header.body.Results[%d].(*output).body.Source is %v, expected %v", i, newOut.body.Source.Value, oldOut.AssetAmount)
				}
				if newOut.body.ControlProgram.VMVersion != 1 {
					t.Errorf("header.body.Results[%d].(*output).body.ControlProgram.VMVersion is %d, expected 1", i, newOut.body.ControlProgram.VMVersion)
				}
				if !bytes.Equal(newOut.body.ControlProgram.Code, oldOut.ControlProgram) {
					t.Errorf("header.body.Results[%d].(*output).body.ControlProgram.Code is %x, expected %x", i, newOut.body.ControlProgram.Code, oldOut.ControlProgram)
				}
				if (newOut.body.Reference != entryRef{}) {
					t.Errorf("header.body.Results[%d].(*output).body.Reference is %x, expected zero", i, newOut.body.Reference[:])
				}
				if (newOut.body.ExtHash != extHash{}) {
					t.Errorf("header.body.Results[%d].(*output).body.ExtHash is %x, expected zero", i, newOut.body.ExtHash[:])
				}
			} else {
				t.Errorf("header.body.Results[%d] has type %s, expected output1", i, resultEntry.Type())
			}
		} else {
			t.Errorf("entryMap contains nothing for header.body.Results[%d] (%x)", i, header.body.Results[i][:])
		}
	}
}

func mustDecodeHash(hash string) (h [32]byte) {
	if len(hash) != hex.EncodedLen(len(h)) {
		panic("wrong length hash")
	}
	_, err := hex.Decode(h[:], []byte(hash))
	if err != nil {
		panic(err)
	}
	return h
}
