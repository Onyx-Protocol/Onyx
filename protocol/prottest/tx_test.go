package prottest

import (
	"testing"

	"chain/protocol/validation"
)

func TestNewIssuance(t *testing.T) {
	err := validation.CheckTxWellFormed(NewIssuanceTx(t, NewChain(t)).TxEntries)
	if err != nil {
		t.Error(err)
	}
}
