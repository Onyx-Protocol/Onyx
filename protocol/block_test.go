package protocol

import (
	"context"
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/mempool"
	"chain/protocol/memstore"
	"chain/protocol/state"
	"chain/testutil"
)

func TestGetBlock(t *testing.T) {
	ctx := context.Background()

	b1 := &bc.Block{BlockHeader: bc.BlockHeader{Height: 1}}
	emptyPool := mempool.New()
	noBlocks := memstore.New()
	oneBlock := memstore.New()
	oneBlock.SaveBlock(ctx, b1)
	oneBlock.SaveSnapshot(ctx, 1, state.Empty())

	cases := []struct {
		store   Store
		want    *bc.Block
		wantErr bool
	}{
		{noBlocks, nil, true},
		{oneBlock, b1, false},
	}

	for _, test := range cases {
		c, err := NewChain(ctx, b1.Hash(), test.store, emptyPool, nil)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		got, gotErr := c.GetBlock(ctx, c.Height())
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got latest = %+v want %+v", got, test.want)
		}
		if (gotErr != nil) != test.wantErr {
			t.Errorf("got latest err = %q want err?: %t", gotErr, test.wantErr)
		}
	}
}

func TestNoTimeTravel(t *testing.T) {
	ctx := context.Background()
	c, err := NewChain(ctx, bc.Hash{}, memstore.New(), mempool.New(), nil)
	if err != nil {
		t.Fatal(err)
	}

	c.setHeight(1)
	c.setHeight(2)

	c.setHeight(1) // don't go backward
	if c.state.height != 2 {
		t.Fatalf("c.state.height = %d want 2", c.state.height)
	}
}

func TestWaitForBlockSoonAlreadyExists(t *testing.T) {
	c, _ := newTestChain(t, time.Now())
	makeEmptyBlock(t, c) // height=2
	makeEmptyBlock(t, c) // height=3

	err := <-c.BlockSoonWaiter(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWaitForBlockSoonDistantFuture(t *testing.T) {
	c, _ := newTestChain(t, time.Now())

	got := <-c.BlockSoonWaiter(context.Background(), 100) // distant future
	want := ErrTheDistantFuture
	if got != want {
		t.Errorf("BlockSoonWaiter(100) = %+v want %+v", got, want)
	}
}

func TestWaitForBlockSoonWaits(t *testing.T) {
	// This test is inherently racy. It's possible
	// that the block creation might run before
	// the wait's internal test loop finds no block.
	// In that case, the test will pass, but it will
	// not have tested anything.
	//
	// It's the best we can do.

	c, _ := newTestChain(t, time.Now())
	makeEmptyBlock(t, c) // height=2

	go func() {
		time.Sleep(10 * time.Millisecond) // sorry for the slow test 😔
		makeEmptyBlock(t, c)              // height=3
	}()

	err := <-c.BlockSoonWaiter(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if g := c.Height(); g != 3 {
		t.Errorf("height after waiting = %d want 3", g)
	}
}

func TestWaitForBlockSoonTimesout(t *testing.T) {
	c, _ := newTestChain(t, time.Now())
	go func() {
		makeEmptyBlock(t, c) // height=2
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := <-c.BlockSoonWaiter(ctx, 3)
	if err != ctx.Err() {
		t.Fatalf("expected timeout err, got %v", err)
	}
}

func TestGenerateBlock(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(233400000, 0)
	c, b1 := newTestChain(t, now)

	initialBlockHash := b1.Hash()
	assetID := bc.ComputeAssetID(nil, initialBlockHash, 1)

	txs := []*bc.Tx{
		bc.NewTx(bc.TxData{
			Version: 1,
			Inputs: []*bc.TxInput{
				bc.NewIssuanceInput(nil, 50, nil, initialBlockHash, nil, [][]byte{
					nil,
					mustDecodeHex("30450221009037e1d39b7d59d24eba8012baddd5f4ab886a51b46f52b7c479ddfa55eeb5c5022076008409243475b25dfba6db85e15cf3d74561a147375941e4830baa69769b5101"),
					mustDecodeHex("51210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae")}),
			},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput(assetID, 50, mustDecodeHex("a9145881cd104f8d64635751ac0f3c0decf9150c110687"), nil),
			},
		}),
		bc.NewTx(bc.TxData{
			Version: 1,
			Inputs: []*bc.TxInput{
				bc.NewIssuanceInput(nil, 50, nil, initialBlockHash, nil, [][]byte{
					nil,
					mustDecodeHex("3045022100f3bcffcfd6a1ce9542b653500386cd0ee7b9c86c59390ca0fc0238c0ebe3f1d6022065ac468a51a016842660c3a616c99a9aa5109a3bad1877ba3e0f010f3972472e01"),
					mustDecodeHex("51210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae"),
				}),
			},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput(assetID, 50, mustDecodeHex("a914c171e443e05b953baa7b7d834028ed91e47b4d0b87"), nil),
			},
		}),
	}
	for _, tx := range txs {
		err := c.pool.Insert(ctx, tx)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
	}

	got, _, err := c.GenerateBlock(ctx, b1, state.Empty(), now)
	if err != nil {
		t.Fatalf("err got = %v want nil", err)
	}

	// TODO(bobg): verify these hashes are correct
	var wantTxRoot, wantAssetsRoot bc.Hash
	copy(wantTxRoot[:], mustDecodeHex("d0e593c846d7b189bd3e2f55e680016b14989329af1c5e388ff246caedf04bd3"))
	copy(wantAssetsRoot[:], mustDecodeHex("903d9a10ece41f86b7c2cf23c25b09c2086b321d6d63e2ec7fc7405f84121542"))

	want := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:                bc.NewBlockVersion,
			Height:                 2,
			PreviousBlockHash:      b1.Hash(),
			TransactionsMerkleRoot: wantTxRoot,
			AssetsMerkleRoot:       wantAssetsRoot,
			TimestampMS:            bc.Millis(now),
			ConsensusProgram:       b1.ConsensusProgram,
		},
		Transactions: txs,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("generated block:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestValidateBlockForSig(t *testing.T) {
	initialBlock, err := NewInitialBlock(testutil.TestPubs, 1, time.Now())
	if err != nil {
		t.Fatal("unexpected error ", err)
	}

	ctx := context.Background()
	c, err := NewChain(ctx, initialBlock.Hash(), memstore.New(), mempool.New(), nil)
	if err != nil {
		t.Fatal("unexpected error ", err)
	}

	err = c.ValidateBlockForSig(ctx, initialBlock)
	if err != nil {
		t.Error("unexpected error ", err)
	}
}

// newTestChain returns a new Chain using memstore and mempool for storage,
// along with an initial block b1 (with a 0/0 multisig program).
// It commits b1 before returning.
func newTestChain(tb testing.TB, ts time.Time) (c *Chain, b1 *bc.Block) {
	ctx := context.Background()

	var err error

	b1, err = NewInitialBlock(nil, 0, ts)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	c, err = NewChain(ctx, b1.Hash(), memstore.New(), mempool.New(), nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	// TODO(tessr): consider adding MaxIssuanceWindow to NewChain
	c.MaxIssuanceWindow = 48 * time.Hour
	err = c.CommitBlock(ctx, b1, state.Empty())
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	return c, b1
}

func makeEmptyBlock(tb testing.TB, c *Chain) {
	ctx := context.Background()

	curBlock, err := c.GetBlock(ctx, c.Height())
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	if len(curBlock.Transactions) > 0 {
		tb.Fatal("cannot make nonempty block")
	}

	curState := state.Empty()

	nextBlock, nextState, err := c.GenerateBlock(ctx, curBlock, curState, time.Now())
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	err = c.CommitBlock(ctx, nextBlock, nextState)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
}

func mustDecodeHex(s string) []byte {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}
