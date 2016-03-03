package fedchain

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/fedtest"
	"chain/fedchain/memstore"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
	"chain/testutil"
)

func TestIdempotentAddTx(t *testing.T) {
	ctx, fc := newContextFC(t)

	issueTx, _, _ := fedtest.Issue(t, nil, nil, 1)

	for i := 0; i < 2; i++ {
		err := fc.AddTx(ctx, issueTx)
		if err != nil {
			testutil.FatalErr(t, err)
		}
	}

	// still idempotent after block lands
	err := fc.AddBlock(ctx, &bc.Block{})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	block, _, err := fc.GenerateBlock(ctx, time.Now())
	block.SignatureScript = []byte{txscript.OP_TRUE}
	err = fc.AddBlock(ctx, block)
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
	store := memstore.New()
	fc, err := New(ctx, store, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	issueTx, _, dest1 := fedtest.Issue(t, nil, nil, 1)
	err = fc.AddTx(ctx, issueTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	transferTx := fedtest.Transfer(t, &state.Output{
		TxOutput: *issueTx.Outputs[0],
		Outpoint: bc.Outpoint{Hash: issueTx.Hash, Index: 0},
	}, dest1, fedtest.Dest(t))
	err = fc.AddTx(ctx, transferTx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	invalidTransfer := fedtest.Transfer(t, &state.Output{
		TxOutput: *issueTx.Outputs[0],
		Outpoint: bc.Outpoint{Hash: issueTx.Hash, Index: 0},
	}, dest1, fedtest.Dest(t))

	err = fc.AddTx(ctx, invalidTransfer)
	if errors.Root(err) != ErrTxRejected {
		t.Fatalf("got err = %q want %q", errors.Root(err), ErrTxRejected)
	}
}
