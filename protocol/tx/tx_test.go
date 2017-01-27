package tx

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/protocol/bc"
	"chain/testutil"
)

func TestMapTx(t *testing.T) {
	// sample data copied from protocol/bc/transaction_test.go

	oldTx := sampleTx()
	oldOuts := oldTx.Outputs

	headerEntry, entryMap, err := mapTx(oldTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	t.Log(spew.Sdump(entryMap))

	header := headerEntry.body.(*header)

	if header.Version != 1 {
		t.Errorf("header.Version is %d, expected 1", header.Version)
	}
	if header.MinTimeMS != oldTx.MinTime {
		t.Errorf("header.MinTimeMS is %d, expected %d", header.MinTimeMS, oldTx.MinTime)
	}
	if header.MaxTimeMS != oldTx.MaxTime {
		t.Errorf("header.MaxTimeMS is %d, expected %d", header.MaxTimeMS, oldTx.MaxTime)
	}
	if len(header.Results) != len(oldOuts) {
		t.Errorf("header.Results contains %d item(s), expected %d", len(header.Results), len(oldOuts))
	}

	for i, oldOut := range oldOuts {
		if resultEntry, ok := entryMap[header.Results[i]]; ok {
			if newOut, ok := resultEntry.body.(*output); ok {
				if newOut.Source.Value != oldOut.AssetAmount {
					t.Errorf("header.Results[%d].Source is %v, expected %v", i, newOut.Source.Value, oldOut.AssetAmount)
				}
				if newOut.ControlProgram.VMVersion != 1 {
					t.Errorf("header.Results[%d].ControlProgram.VMVersion is %d, expected 1", i, newOut.ControlProgram.VMVersion)
				}
				if !bytes.Equal(newOut.ControlProgram.Code, oldOut.ControlProgram) {
					t.Errorf("header.Results[%d].ControlProgram.Code is %x, expected %x", i, newOut.ControlProgram.Code, oldOut.ControlProgram)
				}
				if (newOut.Data != entryRef{}) {
					t.Errorf("header.Results[%d].Reference is %x, expected zero", i, newOut.Data[:])
				}
				if (newOut.ExtHash != extHash{}) {
					t.Errorf("header.Results[%d].ExtHash is %x, expected zero", i, newOut.ExtHash[:])
				}
			} else {
				t.Errorf("header.Results[%d] has type %s, expected output1", i, resultEntry.Type())
			}
		} else {
			t.Errorf("entryMap contains nothing for header.Results[%d] (%x)", i, header.Results[i][:])
		}
	}
}

func BenchmarkHashEmptyTx(b *testing.B) {
	tx := &bc.TxData{}
	for i := 0; i < b.N; i++ {
		_, err := HashTx(tx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashNonemptyTx(b *testing.B) {
	tx := sampleTx()
	for i := 0; i < b.N; i++ {
		_, err := HashTx(tx)
		if err != nil {
			b.Fatal(err)
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

func sampleTx() *bc.TxData {
	assetID := bc.ComputeAssetID([]byte{1}, mustDecodeHash("03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"), 1, bc.EmptyStringHash)
	return &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(bc.ComputeOutputID(mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), 0), nil, assetID, 1000000000000, []byte{1}, []byte("input")),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 600000000000, []byte{1}, nil),
			bc.NewTxOutput(assetID, 400000000000, []byte{2}, nil),
		},
		MinTime:       1492590000,
		MaxTime:       1492590591,
		ReferenceData: []byte("distribution"),
	}
}
