package protocol

import (
	"context"
	"encoding/hex"
	"reflect"
	"sync"
	"testing"
	"time"

	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest/memstore"
	"chain/protocol/state"
	"chain/testutil"
)

func TestGetBlock(t *testing.T) {
	ctx := context.Background()

	b1 := &legacy.Block{BlockHeader: legacy.BlockHeader{Height: 1}}
	noBlocks := memstore.New()
	oneBlock := memstore.New()
	oneBlock.SaveBlock(ctx, b1)
	oneBlock.SaveSnapshot(ctx, 1, state.Empty())

	cases := []struct {
		store   Store
		want    *legacy.Block
		wantErr bool
	}{
		{noBlocks, nil, true},
		{oneBlock, b1, false},
	}

	for _, test := range cases {
		c, err := NewChain(ctx, b1.Hash(), test.store, nil)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		got, gotErr := c.GetBlock(ctx, c.Height())
		if !testutil.DeepEqual(got, test.want) {
			t.Errorf("got latest = %+v want %+v", got, test.want)
		}
		if (gotErr != nil) != test.wantErr {
			t.Errorf("got latest err = %q want err?: %t", gotErr, test.wantErr)
		}
	}
}

func TestNoTimeTravel(t *testing.T) {
	ctx := context.Background()
	c, err := NewChain(ctx, bc.Hash{}, memstore.New(), nil)
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
		time.Sleep(10 * time.Millisecond) // sorry for the slow test 
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
	assetID := bc.ComputeAssetID(nil, &initialBlockHash, 1, &bc.EmptyStringHash)

	txs := []*legacy.Tx{
		legacy.NewTx(legacy.TxData{
			Version: 1,
			MinTime: 233400000000,
			MaxTime: 233400000001,
			Inputs: []*legacy.TxInput{
				legacy.NewIssuanceInput([]byte{1}, 50, nil, initialBlockHash, nil, [][]byte{
					nil,
					mustDecodeHex("30450221009037e1d39b7d59d24eba8012baddd5f4ab886a51b46f52b7c479ddfa55eeb5c5022076008409243475b25dfba6db85e15cf3d74561a147375941e4830baa69769b5101"),
					mustDecodeHex("51210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae")}, nil),
			},
			Outputs: []*legacy.TxOutput{
				legacy.NewTxOutput(assetID, 50, mustDecodeHex("a9145881cd104f8d64635751ac0f3c0decf9150c110687"), nil),
			},
		}),
		legacy.NewTx(legacy.TxData{
			Version: 1,
			MinTime: 233400000000,
			MaxTime: 233400000001,
			Inputs: []*legacy.TxInput{
				legacy.NewIssuanceInput([]byte{2}, 50, nil, initialBlockHash, nil, [][]byte{
					nil,
					mustDecodeHex("3045022100f3bcffcfd6a1ce9542b653500386cd0ee7b9c86c59390ca0fc0238c0ebe3f1d6022065ac468a51a016842660c3a616c99a9aa5109a3bad1877ba3e0f010f3972472e01"),
					mustDecodeHex("51210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae"),
				}, nil),
			},
			Outputs: []*legacy.TxOutput{
				legacy.NewTxOutput(assetID, 50, mustDecodeHex("a914c171e443e05b953baa7b7d834028ed91e47b4d0b87"), nil),
			},
		}),
	}

	got, _, err := c.GenerateBlock(ctx, b1, state.Empty(), now, txs)
	if err != nil {
		t.Fatalf("err got = %v want nil", err)
	}

	// TODO(bobg): verify these hashes are correct
	wantTxRoot := mustDecodeHash("ab5f5f111beb1e6b49da8334360589c7da3aac1cdd61067ea9a55bec47cb745c")
	wantAssetsRoot := mustDecodeHash("a31a9b5f71a6d6fa0c87361db4a98c9a82f603f9d9ff584f6613b9d56ccf5ebd")

	want := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:           1,
			Height:            2,
			PreviousBlockHash: b1.Hash(),
			TimestampMS:       bc.Millis(now),
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: wantTxRoot,
				AssetsMerkleRoot:       wantAssetsRoot,
				ConsensusProgram:       b1.ConsensusProgram,
			},
		},
		Transactions: txs,
	}

	if !testutil.DeepEqual(got, want) {
		t.Errorf("generated block:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestValidateBlockForSig(t *testing.T) {
	initialBlock, err := NewInitialBlock(testutil.TestPubs, 1, time.Now())
	if err != nil {
		t.Fatal("unexpected error ", err)
	}

	ctx := context.Background()
	c, err := NewChain(ctx, initialBlock.Hash(), memstore.New(), nil)
	if err != nil {
		t.Fatal("unexpected error ", err)
	}

	err = c.ValidateBlockForSig(ctx, initialBlock)
	if err != nil {
		t.Error("unexpected error ", err)
	}
}

func TestCommitBlockIdempotence(t *testing.T) {
	const numOfBlocks = 10
	const concurrency = 5
	ctx := context.Background()

	now := time.Now()
	c, b1 := newTestChain(t, now)

	var blocks []*legacy.Block
	b, s := b1, state.Empty()
	for i := 0; i < numOfBlocks; i++ {
		tx, _, _ := issue(t, nil, nil, 1)
		newBlock, newSnapshot, err := c.GenerateBlock(ctx, b, s, now.Add(time.Duration(i+1)*time.Second), []*legacy.Tx{tx})
		if err != nil {
			testutil.FatalErr(t, err)
		}
		err = c.CommitAppliedBlock(ctx, newBlock, newSnapshot)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		blocks = append(blocks, newBlock)
		b, s = newBlock, newSnapshot
	}
	wantBlock, wantSnapshot := b, s

	// Create a fresh Chain for the same blockchain / initial hash.
	c, err := NewChain(ctx, b1.Hash(), memstore.New(), nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	c.MaxIssuanceWindow = 48 * time.Hour
	c.setState(b1, state.Empty())

	// Apply all of the blocks concurrently in separate goroutines
	// using CommitBlock. They should all succeed.
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < len(blocks); j++ {
				err := c.CommitBlock(ctx, blocks[j])
				if err != nil {
					testutil.FatalErr(t, err)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	gotBlock, gotSnapshot := c.State()
	if !reflect.DeepEqual(gotBlock, wantBlock) {
		t.Errorf("got block %#v, want %#v", gotBlock, wantBlock)
	}
	if !reflect.DeepEqual(gotSnapshot, wantSnapshot) {
		t.Errorf("got block %#v, want %#v", gotSnapshot, wantSnapshot)
	}
}

// newTestChain returns a new Chain using memstore for storage,
// along with an initial block b1 (with a 0/0 multisig program).
// It commits b1 before returning.
func newTestChain(tb testing.TB, ts time.Time) (c *Chain, b1 *legacy.Block) {
	ctx := context.Background()

	var err error

	b1, err = NewInitialBlock(nil, 0, ts)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	c, err = NewChain(ctx, b1.Hash(), memstore.New(), nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	// TODO(tessr): consider adding MaxIssuanceWindow to NewChain
	c.MaxIssuanceWindow = 48 * time.Hour
	err = c.CommitAppliedBlock(ctx, b1, state.Empty())
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

	nextBlock, nextState, err := c.GenerateBlock(ctx, curBlock, curState, time.Now(), nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	err = c.CommitAppliedBlock(ctx, nextBlock, nextState)
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

func mustDecodeHash(s string) (h bc.Hash) {
	err := h.UnmarshalText([]byte(s))
	if err != nil {
		panic(err)
	}
	return h
}
