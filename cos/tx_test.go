package cos

import (
	"context"
	"testing"
	"time"

	"chain/cos/fedtest"
	"chain/cos/mempool"
	"chain/cos/memstore"
	"chain/cos/state"
	"chain/cos/txscript"
	"chain/testutil"
)

func TestIdempotentAddTx(t *testing.T) {
	ctx, fc := newContextFC(t)
	genesis, err := fc.UpsertGenesisBlock(ctx, nil, 0, time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	issueTx, _, _ := fedtest.Issue(t, nil, nil, 1)

	err = fc.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// still idempotent after block lands
	block, err := fc.GenerateBlock(ctx, genesis, state.Empty(), time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}
	block.Witness = [][]byte{{txscript.OP_0}}
	tree, err := fc.ValidateBlock(ctx, state.Empty(), genesis, block)
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
	fc, err := NewFC(ctx, memstore.New(), mempool.New(), nil, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	_, err = fc.UpsertGenesisBlock(ctx, nil, 0, time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	issueTx, _, dest1 := fedtest.Issue(t, nil, nil, 1)
	err = fc.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	transferTx := fedtest.Transfer(t, fedtest.StateOut(issueTx, 0), dest1, fedtest.Dest(t))

	err = fc.AddTx(ctx, transferTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}
