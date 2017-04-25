package protocol_test

import (
	"testing"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
)

func TestNoncelessIssuance(t *testing.T) {
	c := prottest.NewChain(t)
	tx := prottest.NewIssuanceTx(t, c, func(tx *legacy.Tx) {
		// Remove the issuance nonce.
		tx.Inputs[0].TypedInput.(*legacy.IssuanceInput).Nonce = nil
	})

	err := c.ValidateTx(legacy.MapTx(&tx.TxData))
	if errors.Root(err) != bc.ErrMissingEntry {
		t.Fatalf("got %s, want %s", err, bc.ErrMissingEntry)
	}
}
