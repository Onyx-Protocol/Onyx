package validation_test

import (
	"context"
	"testing"
	"time"

	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/protocol/state"
	"chain/protocol/validation"
)

func BenchmarkValidateBlock(b *testing.B) {
	b.StopTimer()
	ctx := context.Background()

	c := prottest.NewChain(b)
	b1, s := c.State()

	// Generate a large block to validate.
	for i := 0; i < 1000; i++ {
		err := c.AddTx(ctx, prottest.NewIssuanceTx(b, c))
		if err != nil {
			b.Fatal(err)
		}
	}
	nextBlock, _, err := c.GenerateBlock(ctx, b1, s, time.Now())
	if err != nil {
		b.Fatal(err)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		st := state.Copy(s)
		err := validation.ValidateBlockForAccept(ctx, st, b1.Hash(), b1, nextBlock, validation.CheckTxWellFormed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCalcMerkleRoot(b *testing.B) {
	b.StopTimer()
	c := prottest.NewChain(b)
	var txs []*bc.Tx
	for i := 0; i < 5000; i++ {
		txs = append(txs, prottest.NewIssuanceTx(b, c))
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		validation.CalcMerkleRoot(txs)
	}
}
