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

	if header.Body.Version != 1 {
		t.Errorf("header.Body.Version is %d, expected 1", header.Body.Version)
	}
	if header.Body.MinTimeMS != oldTx.MinTime {
		t.Errorf("header.Body.MinTimeMS is %d, expected %d", header.Body.MinTimeMS, oldTx.MinTime)
	}
	if header.Body.MaxTimeMS != oldTx.MaxTime {
		t.Errorf("header.Body.MaxTimeMS is %d, expected %d", header.Body.MaxTimeMS, oldTx.MaxTime)
	}
	if len(header.Body.ResultIDs) != len(oldOuts) {
		t.Errorf("header.Body.ResultIDs contains %d item(s), expected %d", len(header.Body.ResultIDs), len(oldOuts))
	}

	for i, oldOut := range oldOuts {
		if resultEntry, ok := entryMap[header.Body.ResultIDs[i]]; ok {
			if newOut, ok := resultEntry.(*Output); ok {
				if newOut.Body.Source.Value != oldOut.AssetAmount {
					t.Errorf("header.Body.ResultIDs[%d].(*output).Body.Source is %v, expected %v", i, newOut.Body.Source.Value, oldOut.AssetAmount)
				}
				if newOut.Body.ControlProgram.VMVersion != 1 {
					t.Errorf("header.Body.ResultIDs[%d].(*output).Body.ControlProgram.VMVersion is %d, expected 1", i, newOut.Body.ControlProgram.VMVersion)
				}
				if !bytes.Equal(newOut.Body.ControlProgram.Code, oldOut.ControlProgram) {
					t.Errorf("header.Body.ResultIDs[%d].(*output).Body.ControlProgram.Code is %x, expected %x", i, newOut.Body.ControlProgram.Code, oldOut.ControlProgram)
				}
				if newOut.Body.Data != hashData(oldOut.ReferenceData) {
					want := hashData(oldOut.ReferenceData)
					t.Errorf("header.Body.ResultIDs[%d].(*output).Body.Data is %x, expected %x", i, newOut.Body.Data[:], want[:])
				}
				if (newOut.Body.ExtHash != Hash{}) {
					t.Errorf("header.Body.ResultIDs[%d].(*output).Body.ExtHash is %x, expected zero", i, newOut.Body.ExtHash[:])
				}
			} else {
				t.Errorf("header.Body.ResultIDs[%d] has type %s, expected output1", i, resultEntry.Type())
			}
		} else {
			t.Errorf("entryMap contains nothing for header.Body.ResultIDs[%d] (%x)", i, header.Body.ResultIDs[i][:])
		}
	}
}
