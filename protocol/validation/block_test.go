package validation

import (
	"context"
	"testing"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/vm"
)

// emptyMerkleRoot is the SHA3-256 of "".
var emptyMerkleRoot = mustParseHash("a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a")

func TestValidateBlockHeader(t *testing.T) {
	ctx := context.Background()
	prev := bc.MapBlock(&bc.Block{BlockHeader: bc.BlockHeader{
		Height:      1,
		TimestampMS: 5,
		BlockCommitment: bc.BlockCommitment{
			ConsensusProgram: []byte{byte(vm.OP_5), byte(vm.OP_ADD), byte(vm.OP_9), byte(vm.OP_EQUAL)},
		},
	}})
	cases := []struct {
		desc   string
		header bc.BlockHeader
		want   error
	}{{
		desc: "bad prev block hash",
		header: bc.BlockHeader{
			PreviousBlockHash: bc.Hash{},
			Height:            2,
			BlockCommitment: bc.BlockCommitment{
				TransactionsMerkleRoot: emptyMerkleRoot,
			},
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{{0x04}},
			},
		},
		want: ErrBadPrevHash,
	}, {
		desc: "bad block height",
		header: bc.BlockHeader{
			PreviousBlockHash: prev.ID,
			Height:            3,
			BlockCommitment: bc.BlockCommitment{
				TransactionsMerkleRoot: emptyMerkleRoot,
			},
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{{0x04}},
			},
		},
		want: ErrBadHeight,
	}, {
		desc: "bad block timestamp",
		header: bc.BlockHeader{
			PreviousBlockHash: prev.ID,
			Height:            2,
			TimestampMS:       3,
			BlockCommitment: bc.BlockCommitment{
				TransactionsMerkleRoot: emptyMerkleRoot,
			},
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{{0x04}},
			},
		},
		want: ErrBadTimestamp,
	}, {
		desc: "fake initial block",
		header: bc.BlockHeader{
			PreviousBlockHash: prev.ID,
			Height:            1,
			TimestampMS:       6,
			BlockCommitment: bc.BlockCommitment{
				TransactionsMerkleRoot: emptyMerkleRoot,
				ConsensusProgram:       []byte{byte(vm.OP_5), byte(vm.OP_ADD), byte(vm.OP_9), byte(vm.OP_EQUAL)},
			},
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{{0x04}},
			},
		},
		want: ErrBadHeight,
	}, {
		desc: "bad block output script",
		header: bc.BlockHeader{
			PreviousBlockHash: prev.ID,
			Height:            2,
			TimestampMS:       6,
			BlockCommitment: bc.BlockCommitment{
				TransactionsMerkleRoot: emptyMerkleRoot,
				ConsensusProgram:       []byte{byte(vm.OP_FAIL)},
			},
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{{0x04}},
			},
		},
		want: ErrBadScript,
	}, {
		desc: "bad block signature script",
		header: bc.BlockHeader{
			PreviousBlockHash: prev.ID,
			Height:            2,
			TimestampMS:       6,
			BlockCommitment: bc.BlockCommitment{
				TransactionsMerkleRoot: emptyMerkleRoot,
				ConsensusProgram:       []byte{byte(vm.OP_5), byte(vm.OP_ADD), byte(vm.OP_9), byte(vm.OP_EQUAL)},
			},
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{{0x03}},
			},
		},
		want: ErrBadSig,
	}, {
		desc: "valid header",
		header: bc.BlockHeader{
			PreviousBlockHash: prev.ID,
			Height:            2,
			TimestampMS:       6,
			BlockCommitment: bc.BlockCommitment{
				TransactionsMerkleRoot: emptyMerkleRoot,
				ConsensusProgram:       []byte{byte(vm.OP_5), byte(vm.OP_ADD), byte(vm.OP_9), byte(vm.OP_EQUAL)},
			},
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{{0x04}},
			},
		},
		want: nil,
	}}
	for i, c := range cases {
		block := bc.MapBlock(&bc.Block{BlockHeader: c.header})
		snap := state.Empty()
		got := ValidateBlockForAccept(ctx, snap, prev.ID, prev, block, nil) // nil b/c no txs to validate
		if errors.Root(got) != c.want {
			t.Errorf("%d", i)
			t.Errorf("%s: got %q want %q", c.desc, got, c.want)
		}
	}
}
