package prottest

import (
	"testing"

	"chain/protocol/validation"
)

func TestNewIssuance(t *testing.T) {
	c := NewChain(t)
	iss := NewIssuanceTx(t, c)
	err := validation.ValidateTx(iss.TxEntries, c.InitialBlockHash)
	if err != nil {
		t.Error(err)
	}
}
