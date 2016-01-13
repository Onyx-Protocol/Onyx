package assettest

import (
	"testing"

	"chain/fedchain/bc"
)

// ExpectMatchingInputs tests each input in a given tx, expecting to
// find exactly N that satisfy the given predicate.
func ExpectMatchingInputs(t *testing.T, tx *bc.Tx, n int, failMsg string, pred func(*testing.T, *bc.TxInput) bool) {
	var found int
	for _, txInput := range tx.Inputs {
		if pred(t, txInput) {
			found++
		}
	}
	if found != n {
		t.Errorf("ExpectMatchingInputs: got %d match(es), wanted %d: %s", found, n, failMsg)
	}
}

// ExpectMatchingOutputs tests each output in a given tx, expecting to
// find exactly N that satisfy the given predicate.
func ExpectMatchingOutputs(t *testing.T, tx *bc.Tx, n int, failMsg string, pred func(*testing.T, *bc.TxOutput) bool) {
	var found int
	for _, txOutput := range tx.Outputs {
		if pred(t, txOutput) {
			found++
		}
	}
	if found != n {
		t.Errorf("ExpectMatchingOutputs: got %d match(es), wanted %d: %s", found, n, failMsg)
	}
}
