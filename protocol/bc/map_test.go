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
	if header.Body.MinTimeMs != oldTx.MinTime {
		t.Errorf("header.Body.MinTimeMs is %d, expected %d", header.Body.MinTimeMs, oldTx.MinTime)
	}
	if header.Body.MaxTimeMs != oldTx.MaxTime {
		t.Errorf("header.Body.MaxTimeMs is %d, expected %d", header.Body.MaxTimeMs, oldTx.MaxTime)
	}
	if len(header.Body.ResultIds) != len(oldOuts) {
		t.Errorf("header.Body.ResultIds contains %d item(s), expected %d", len(header.Body.ResultIds), len(oldOuts))
	}

	for i, oldOut := range oldOuts {
		if resultEntry, ok := entryMap[header.Body.ResultIds[i].Hash()]; ok {
			if newOut, ok := resultEntry.(*Output); ok {
				if newOut.Body.Source.Value.AssetAmount() != oldOut.AssetAmount {
					t.Errorf("header.Body.ResultIds[%d].(*output).Body.Source is %v, expected %v", i, newOut.Body.Source.Value, oldOut.AssetAmount)
				}
				if newOut.Body.ControlProgram.VmVersion != 1 {
					t.Errorf("header.Body.ResultIds[%d].(*output).Body.ControlProgram.VmVersion is %d, expected 1", i, newOut.Body.ControlProgram.VmVersion)
				}
				if !bytes.Equal(newOut.Body.ControlProgram.Code, oldOut.ControlProgram) {
					t.Errorf("header.Body.ResultIds[%d].(*output).Body.ControlProgram.Code is %x, expected %x", i, newOut.Body.ControlProgram.Code, oldOut.ControlProgram)
				}
				if newOut.Body.Data.Hash() != hashData(oldOut.ReferenceData) {
					want := hashData(oldOut.ReferenceData)
					t.Errorf("header.Body.ResultIds[%d].(*output).Body.Data is %x, expected %x", i, newOut.Body.Data.Hash().Bytes(), want[:])
				}
				if !newOut.Body.ExtHash.IsZero() {
					t.Errorf("header.Body.ResultIds[%d].(*output).Body.ExtHash is %x, expected zero", i, newOut.Body.ExtHash.Hash().Bytes())
				}
			} else {
				t.Errorf("header.Body.ResultIds[%d] has type %s, expected output1", i, resultEntry.Type())
			}
		} else {
			t.Errorf("entryMap contains nothing for header.Body.ResultIds[%d] (%x)", i, header.Body.ResultIds[i].Hash().Bytes())
		}
	}
}
