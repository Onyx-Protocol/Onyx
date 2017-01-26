package prottest

import (
	"testing"

	"chain/protocol/validation"
)

func TestNewIssuance(t *testing.T) {
	err := validation.CheckTxWellFormed(NewIssuanceTx(t, NewChain(t)))
	if err != nil {
		t.Error(err)
	}
}
