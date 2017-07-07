package generator

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"chain/crypto/ed25519"
	"chain/database/pg/pgtest"
	"chain/protocol"
	"chain/protocol/bc/bctest"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
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

	go New(c, nil, dbtx).Generate(ctx, 50*time.Millisecond, func(error) {})

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

func TestGeneratorSignatureFailures(t *testing.T) {
	ctx := context.Background()
	c := prottest.NewChain(t, prottest.WithBlockSigners(1, 1))
	pubkeys, privkeys := prottest.BlockKeyPairs(c)

	// Use a signer that fails to sign the first 3 times then succeeds.
	failuresRemaining := int64(3)
	signers := []BlockSigner{testSigner{
		before: func() error {
			if v := atomic.AddInt64(&failuresRemaining, -1); v >= 0 {
				return fmt.Errorf("error %d", v)
			}
			return nil
		},
		pubKey:  pubkeys[0],
		privKey: privkeys[0],
	}}

	g := New(c, signers, pgtest.NewTx(t))
	tx := bctest.NewIssuanceTx(t, prottest.Initial(t, c).Hash())
	g.pool = append(g.pool, tx)

	height := c.Height()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go g.Generate(ctx, 50*time.Millisecond, func(err error) { t.Logf("%s\n", err) })

	<-c.BlockWaiter(height + 1)
	block, err := c.GetBlock(ctx, height+1)
	if err != nil {
		t.Fatal(err)
	}
	if block.Transactions[0].ID != tx.ID {
		t.Errorf("got tx %s want %s", block.Transactions[0].ID, tx.ID)
	}
}

func TestGetAndAddBlockSignatures(t *testing.T) {
	c := prottest.NewChain(t, prottest.WithBlockSigners(1, 1))
	pubkeys, privkeys := prottest.BlockKeyPairs(c)

	g := New(c, []BlockSigner{testSigner{nil, pubkeys[0], privkeys[0]}}, nil)

	ctx := context.Background()
	tip, snapshot := c.State()

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

// TestGetAndAddBlockSignaturesRace tests a scenario where all necessary
// signatures are obtained quickly, but a slow signer is still signing.
func TestGetAndAddBlockSignaturesRace(t *testing.T) {
	c := prottest.NewChain(t)
	pubkey, privkey, err := ed25519.GenerateKey(nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	g := New(c, []BlockSigner{testSigner{nil, pubkey, privkey}}, nil)

	ctx := context.Background()
	tip, snapshot := c.State()
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
	before  func() error
	pubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey
}

func (s testSigner) SignBlock(ctx context.Context, marshalledBlock []byte) ([]byte, error) {
	if s.before != nil {
		if err := s.before(); err != nil {
			return nil, err
		}
	}

	var b legacy.Block
	err := b.UnmarshalText(marshalledBlock)
	if err != nil {
		return nil, err
	}
	return ed25519.Sign(s.privKey, b.Hash().Bytes()), nil
}

func (s testSigner) String() string {
	return "test-signer"
}
