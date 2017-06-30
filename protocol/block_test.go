package protocol_test

import (
	"context"
	"testing"
	"time"

	"chain/protocol/bc/bcvm"
	"chain/protocol/bc/bcvm/bcvmtest"
	"chain/protocol/prottest"
)

func TestBCVM(t *testing.T) {
	c := prottest.NewChain(t)

	issuance := bcvmtest.NewIssuanceTx(t)

	err := c.ValidateTx(issuance)
	if err != nil {
		t.Fatal(err)
	}

	curBlock, curState := c.State()

	nextBlock, _, err := c.GenerateBlock(context.Background(), curBlock, curState, time.Now(), [][]byte{issuance})
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	err = c.ValidateBlock(nextBlock, curBlock)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	state, err := c.ApplyValidBlock(nextBlock)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	tx, err := bcvm.NewTx(issuance)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if !state.Tree.Contains(tx.Outputs[0].ID.Bytes()) {
		t.Fatal("expected output in state tree")
	}
}
