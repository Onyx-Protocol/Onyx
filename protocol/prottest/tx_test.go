package prottest

import (
	"testing"

	"chain/protocol/bc"
)

func TestNewIssuance(t *testing.T) {
	c := NewChain(t)
	iss := NewIssuanceTx(t, c)
	err := bc.ValidateTx(iss.TxEntries, c.InitialBlockHash)
	if err != nil {
		t.Error(err)
	}
}
