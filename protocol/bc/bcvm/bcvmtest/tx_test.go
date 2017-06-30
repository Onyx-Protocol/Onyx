package bcvmtest

import (
	"chain/protocol/txvm"
	"testing"
)

func TestNewIssuanceTx(t *testing.T) {
	tx := NewIssuanceTx(t)

	_, ok := txvm.Validate(tx)
	if !ok {
		t.Fatal("expected ok")
	}
}
