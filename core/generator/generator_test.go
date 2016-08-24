package generator

import (
	"context"
	"testing"
	"time"

	"chain/core/blocksigner"
	"chain/core/mockhsm"
	"chain/database/pg/pgtest"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/mempool"
	"chain/protocol/memstore"
	"chain/protocol/state"
	"chain/protocol/validation"
	"chain/protocol/vm"
	"chain/testutil"
)

// newTestFC returns a new FC using memstore and mempool for storage,
// along with an initial block b1 (with a 0/0 multisig program).
// It commits b1 before returning.
func newTestFC(tb testing.TB, ts time.Time) (fc *protocol.FC, b1 *bc.Block) {
	ctx := context.Background()
	fc, err := protocol.NewFC(ctx, memstore.New(), mempool.New(), nil, nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	b1, err = protocol.NewGenesisBlock(nil, 0, ts)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	err = fc.CommitBlock(ctx, b1, state.Empty())
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	return fc, b1
}

func TestGetAndAddBlockSignatures(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	fc, b1 := newTestFC(t, time.Now())

	// TODO(kr): tweak the generator's design to not
	// take a hard dependency on mockhsm.
	// See also similar comment in $CHAIN/core/blocksigner/blocksigner.go.
	hsm := mockhsm.New(dbtx)
	xpub, err := hsm.CreateKey(ctx, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	localSigner := blocksigner.New(xpub.XPub, hsm, dbtx, fc)
	config := Config{
		LocalSigner: localSigner,
		FC:          fc,
	}

	g := New(b1, state.Empty(), config)

	tip, snapshot, err := fc.Recover(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block, err := fc.GenerateBlock(ctx, tip, snapshot, time.Now())
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
