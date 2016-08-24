package cos

import (
	"context"
	"testing"
	"time"

	"chain/cos/fedtest"
	"chain/cos/state"
	"chain/testutil"
)

func TestIdempotentAddTx(t *testing.T) {
	ctx := context.Background()
	fc, b1 := newTestFC(t, time.Now())

	issueTx, _, _ := fedtest.Issue(t, nil, nil, 1)

	err := fc.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// still idempotent after block lands
	block, err := fc.GenerateBlock(ctx, b1, state.Empty(), time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	tree, err := fc.ValidateBlock(ctx, state.Empty(), b1, block)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = fc.CommitBlock(ctx, block, tree)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = fc.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func TestAddTx(t *testing.T) {
	ctx := context.Background()
	fc, _ := newTestFC(t, time.Now())

	issueTx, _, dest1 := fedtest.Issue(t, nil, nil, 1)
	err := fc.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	transferTx := fedtest.Transfer(t, fedtest.StateOut(issueTx, 0), dest1, fedtest.Dest(t))

	err = fc.AddTx(ctx, transferTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}
