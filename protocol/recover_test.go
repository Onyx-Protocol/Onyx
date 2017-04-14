package protocol

import (
	"context"
	"log"
	"testing"
	"time"

	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest/memstore"
	"chain/protocol/state"
	"chain/testutil"
)

func TestRecoverSnapshotNoAdditionalBlocks(t *testing.T) {
	store := memstore.New()
	b, err := NewInitialBlock(nil, 0, time.Now().Add(-time.Minute))
	if err != nil {
		testutil.FatalErr(t, err)
	}
	c1, err := NewChain(context.Background(), b.Hash(), store, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = c1.CommitAppliedBlock(context.Background(), b, state.Empty())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Snapshots are applied asynchronously. This loops waits
	// until the snapshot is created.
	for {
		_, height, _ := store.LatestSnapshot(context.Background())
		if height > 0 {
			break
		}
	}

	ctx := context.Background()

	c2, err := NewChain(context.Background(), b.Hash(), store, nil)
	if err != nil {
		t.Fatal(err)
	}
	block, snapshot, err := c2.Recover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if block.Height != 1 {
		t.Fatalf("block.Height = %d, want %d", block.Height, 1)
	}

	err = c2.ValidateBlockForSig(ctx, createEmptyBlock(block, snapshot))
	if err != nil {
		t.Fatal(err)
	}
}

func createEmptyBlock(block *legacy.Block, snapshot *state.Snapshot) *legacy.Block {
	root, err := bc.MerkleRoot(nil)
	if err != nil {
		log.Fatalf("calculating empty merkle root: %s", err)
	}

	return &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:           1,
			Height:            block.Height + 1,
			PreviousBlockHash: block.Hash(),
			TimestampMS:       bc.Millis(time.Now()),
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: root,
				AssetsMerkleRoot:       snapshot.Tree.RootHash(),
				ConsensusProgram:       block.ConsensusProgram,
			},
		},
	}
}
