package validation

import (
	"testing"

	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
)

func TestFuzzAssetIdNilPointer(t *testing.T) {
	const (
		blockchainID = `50935a092ffad7ec9fbac4f4486db6c3b8cd5b9f51cf697248584dde286a7220`
		input        = `07300730303030303030000001302b3030303030303030303030303030303030303030303030303030303030303030303030303030303030303000253030303030303030303030303030303030303030303030303030303030303030303030303000`
	)

	var testBlockchainID bc.Hash
	err := testBlockchainID.UnmarshalText([]byte(blockchainID))
	if err != nil {
		t.Fatal(err)
	}

	var tx legacy.Tx
	err = tx.UnmarshalText([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	ValidateTx(tx.Tx, testBlockchainID)
}
