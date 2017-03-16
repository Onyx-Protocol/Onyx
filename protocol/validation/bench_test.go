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

func BenchmarkValidateTx(b *testing.B) {
	c := prottest.NewChain(b)
	tx := prottest.NewIssuanceTx(b, c)
	for i := 0; i < b.N; i++ {
		err := validation.CheckTxWellFormed(tx.TxEntries)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateBlock(b *testing.B) {
	b.StopTimer()
	ctx := context.Background()

	c := prottest.NewChain(b)
	b1, s := c.State()

	// Generate a large block to validate.
	var txs []*bc.Tx
	for i := 0; i < 1000; i++ {
		txs = append(txs, prottest.NewIssuanceTx(b, c))
	}

	nextBlock, _, err := c.GenerateBlock(ctx, b1, s, time.Now(), txs)
	if err != nil {
		b.Fatal(err)
	}

	b1Entries := bc.MapBlock(b1)
	nextBlockEntries := bc.MapBlock(nextBlock)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		st := state.Copy(s)
		err := validation.ValidateBlockForAccept(ctx, st, b1Entries.ID, b1Entries, nextBlockEntries, validation.CheckTxWellFormed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCalcMerkleRoot(b *testing.B) {
	b.StopTimer()
	c := prottest.NewChain(b)
	var txs []*bc.TxEntries
	for i := 0; i < 5000; i++ {
		txs = append(txs, prottest.NewIssuanceTx(b, c).TxEntries)
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		validation.CalcMerkleRoot(txs)
	}
}
