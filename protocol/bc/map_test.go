package bc

import (
	"bytes"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/testutil"
)

func TestMapTx(t *testing.T) {
	// sample data copied from transaction_test.go
	// TODO(bobg): factor out into reusable test utility

	oldTx := sampleTx()
	oldOuts := oldTx.Outputs

	_, header, entryMap, err := mapTx(oldTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	t.Log(spew.Sdump(entryMap))

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
				if newOut.body.Data != hashData(oldOut.ReferenceData) {
					want := hashData(oldOut.ReferenceData)
					t.Errorf("header.body.Results[%d].(*output).body.Data is %x, expected %x", i, newOut.body.Data[:], want[:])
				}
				if (newOut.body.ExtHash != Hash{}) {
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
