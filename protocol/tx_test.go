package protocol

import (
	"context"
	"testing"
	"time"

	"chain/protocol/fedtest"
	"chain/protocol/state"
	"chain/testutil"
)

func TestIdempotentAddTx(t *testing.T) {
	ctx := context.Background()
	c, b1 := newTestChain(t, time.Now())

	issueTx, _, _ := fedtest.Issue(t, nil, nil, 1)

	err := c.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// still idempotent after block lands
	block, err := c.GenerateBlock(ctx, b1, state.Empty(), time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	tree, err := c.ValidateBlock(ctx, state.Empty(), b1, block)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = c.CommitBlock(ctx, block, tree)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = c.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func TestAddTx(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestChain(t, time.Now())

	issueTx, _, dest1 := fedtest.Issue(t, nil, nil, 1)
	err := c.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	transferTx := fedtest.Transfer(t, fedtest.StateOut(issueTx, 0), dest1, fedtest.Dest(t))

	err = c.AddTx(ctx, transferTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}
