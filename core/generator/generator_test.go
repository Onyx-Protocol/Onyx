package generator

import (
	"context"
	"testing"
	"time"

	"chain/core/blocksigner"
	"chain/core/mockhsm"
	"chain/database/pg/pgtest"
	"chain/protocol"
	"chain/protocol/prottest"
	"chain/protocol/state"
	"chain/protocol/validation"
	"chain/protocol/vm"
	"chain/testutil"
)

func TestGetAndAddBlockSignatures(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	c := prottest.NewChain(t)
	b1, err := c.GetBlock(ctx, 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// TODO(kr): tweak the generator's design to not
	// take a hard dependency on mockhsm.
	// See also similar comment in $CHAIN/core/blocksigner/blocksigner.go.
	hsm := mockhsm.New(dbtx)
	xpub, err := hsm.CreateKey(ctx, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	localSigner := blocksigner.New(xpub.XPub, hsm, dbtx, c)
	config := Config{
		LocalSigner: localSigner,
		Chain:       c,
	}

	g := New(b1, state.Empty(), config)

	tip, snapshot, err := c.Recover(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block, err := c.GenerateBlock(ctx, tip, snapshot, time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = g.GetAndAddBlockSignatures(ctx, block, tip)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	ok, err := vm.VerifyBlockHeader(block, tip)
	if err == nil && !ok {
		err = validation.ErrFalseVMResult
	}
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func TestGetAndAddBlockSignaturesInitialBlock(t *testing.T) {
	ctx := context.Background()

	g := new(Generator)
	block, err := protocol.NewGenesisBlock(testutil.TestPubs, 1, time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = g.GetAndAddBlockSignatures(ctx, block, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if len(block.Witness) != 0 {
		t.Fatalf("GetAndAddBlockSignatures produced witness %v, want empty", block.Witness)
	}
}
