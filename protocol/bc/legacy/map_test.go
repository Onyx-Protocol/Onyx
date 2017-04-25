package legacy

import (
	"bytes"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/protocol/bc"
)

func TestMapTx(t *testing.T) {
	// sample data copied from transaction_test.go
	// TODO(bobg): factor out into reusable test utility

	oldTx := sampleTx()
	oldOuts := oldTx.Outputs

	_, header, entryMap := mapTx(oldTx)
	t.Log(spew.Sdump(entryMap))

	if header.Version != 1 {
		t.Errorf("header.Version is %d, expected 1", header.Version)
	}
	if header.MinTimeMs != oldTx.MinTime {
		t.Errorf("header.MinTimeMs is %d, expected %d", header.MinTimeMs, oldTx.MinTime)
	}
	if header.MaxTimeMs != oldTx.MaxTime {
		t.Errorf("header.MaxTimeMs is %d, expected %d", header.MaxTimeMs, oldTx.MaxTime)
	}
	if len(header.ResultIds) != len(oldOuts) {
		t.Errorf("header.ResultIds contains %d item(s), expected %d", len(header.ResultIds), len(oldOuts))
	}

	for i, oldOut := range oldOuts {
		if resultEntry, ok := entryMap[*header.ResultIds[i]]; ok {
			if newOut, ok := resultEntry.(*bc.Output); ok {
				if *newOut.Source.Value != oldOut.AssetAmount {
					t.Errorf("header.ResultIds[%d].(*output).Source is %v, expected %v", i, newOut.Source.Value, oldOut.AssetAmount)
				}
				if newOut.ControlProgram.VmVersion != 1 {
					t.Errorf("header.ResultIds[%d].(*output).ControlProgram.VMVersion is %d, expected 1", i, newOut.ControlProgram.VmVersion)
				}
				if !bytes.Equal(newOut.ControlProgram.Code, oldOut.ControlProgram) {
					t.Errorf("header.ResultIds[%d].(*output).ControlProgram.Code is %x, expected %x", i, newOut.ControlProgram.Code, oldOut.ControlProgram)
				}
				if *newOut.Data != hashData(oldOut.ReferenceData) {
					want := hashData(oldOut.ReferenceData)
					t.Errorf("header.ResultIds[%d].(*output).Data is %x, expected %x", i, newOut.Data.Bytes(), want.Bytes())
				}
				if !newOut.ExtHash.IsZero() {
					t.Errorf("header.ResultIds[%d].(*output).ExtHash is %x, expected zero", i, newOut.ExtHash.Bytes())
				}
			} else {
				t.Errorf("header.ResultIds[%d] has type %T, expected *Output", i, resultEntry)
			}
		} else {
			t.Errorf("entryMap contains nothing for header.ResultIds[%d] (%x)", i, header.ResultIds[i].Bytes())
		}
	}
}
