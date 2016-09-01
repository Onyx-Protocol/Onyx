package generator

import (
	"context"
	"testing"
	"time"

	"chain/crypto/ed25519"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/protocol/state"
	"chain/protocol/validation"
	"chain/protocol/vm"
	"chain/testutil"
)

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

	signer := testSigner{pubKey, privKey}
	g := &generator{
		chain:          c,
		signers:        []BlockSigner{signer},
		latestBlock:    b1,
		latestSnapshot: state.Empty(),
	}

	tip, snapshot, err := c.Recover(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block, _, err := c.GenerateBlock(ctx, tip, snapshot, time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = g.getAndAddBlockSignatures(ctx, block, tip)
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

	g := new(generator)
	block, err := protocol.NewGenesisBlock(testutil.TestPubs, 1, time.Now())
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

func (s testSigner) PubKey() ed25519.PublicKey {
	return s.pubKey
}

func (s testSigner) SignBlock(ctx context.Context, b *bc.Block) ([]byte, error) {
	hash := b.HashForSig()
	return ed25519.Sign(s.privKey, hash[:]), nil
}

func (s testSigner) String() string {
	return "test-signer"
}
