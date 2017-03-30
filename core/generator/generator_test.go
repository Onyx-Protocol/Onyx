package generator

import (
	"context"
	"testing"
	"time"

	"chain/crypto/ed25519"
	"chain/database/pg/pgtest"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/protocol/state"
	"chain/testutil"
)

func TestGeneratorRecovery(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()
	c := prottest.NewChain(t)
	b, s := c.State()

	// Create a new block and save it to pending blocks to simulate
	// a crash after generating a block but before committing it.
	pendingBlock, _, err := c.GenerateBlock(ctx, b, s, time.Now(), nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = savePendingBlock(ctx, dbtx, pendingBlock)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Start Generate which should notice the pending block and commit it.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go New(c, nil, dbtx).Generate(ctx, time.Second, func(error) {}, b, s)

	// Wait for the block to land, and then make sure it's the same block
	// that was pending before we ran Generate.
	<-c.BlockWaiter(pendingBlock.Height)
	confirmedBlock, err := c.GetBlock(ctx, pendingBlock.Height)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if confirmedBlock.Hash() != pendingBlock.Hash() {
		t.Errorf("got=%s, want=%s", confirmedBlock.Hash(), pendingBlock.Hash())
	}
}

func TestGetAndAddBlockSignatures(t *testing.T) {
	ctx := context.Background()

	c := prottest.NewChain(t)
	b1, err := c.GetBlock(ctx, 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	g := New(c, []BlockSigner{testSigner{pubKey, privKey}}, nil)
	g.latestBlock = b1
	g.latestSnapshot = state.Empty()

	tip, snapshot, err := c.Recover(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block, _, err := c.GenerateBlock(ctx, tip, snapshot, time.Now().Add(time.Minute), nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = g.getAndAddBlockSignatures(ctx, block, tip)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = c.ValidateBlock(block, tip)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func TestGetAndAddBlockSignaturesInitialBlock(t *testing.T) {
	ctx := context.Background()

	g := New(nil, nil, nil)
	block, err := protocol.NewInitialBlock(testutil.TestPubs, 1, time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = g.getAndAddBlockSignatures(ctx, block, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if len(block.Witness) != 0 {
		t.Fatalf("getAndAddBlockSignatures produced witness %v, want empty", block.Witness)
	}
}

type testSigner struct {
	pubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey
}

func (s testSigner) SignBlock(ctx context.Context, b *bc.Block) ([]byte, error) {
	hash := b.Hash()
	return ed25519.Sign(s.privKey, hash[:]), nil
}

func (s testSigner) String() string {
	return "test-signer"
}
