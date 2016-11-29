package validation

import (
	"context"
	"testing"

	"chain-stealth/errors"
	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/state"
	"chain-stealth/protocol/vm"
)

// emptyMerkleRoot is the SHA3-256 of "".
var emptyMerkleRoot = mustParseHash("a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a")

func TestValidateBlockHeader(t *testing.T) {
	ctx := context.Background()
	prev := &bc.Block{BlockHeader: bc.BlockHeader{
		Height:           1,
		TimestampMS:      5,
		ConsensusProgram: []byte{byte(vm.OP_5), byte(vm.OP_ADD), byte(vm.OP_9), byte(vm.OP_EQUAL)},
	}}
	prevHash := prev.Hash()
	cases := []struct {
		desc   string
		header bc.BlockHeader
		want   error
	}{{
		desc: "bad prev block hash",
		header: bc.BlockHeader{
			PreviousBlockHash:      bc.Hash{},
			TransactionsMerkleRoot: emptyMerkleRoot,
			Height:                 2,
			Witness:                [][]byte{{0x04}},
		},
		want: ErrBadPrevHash,
	}, {
		desc: "bad block height",
		header: bc.BlockHeader{
			PreviousBlockHash:      prevHash,
			TransactionsMerkleRoot: emptyMerkleRoot,
			Height:                 3,
			Witness:                [][]byte{{0x04}},
		},
		want: ErrBadHeight,
	}, {
		desc: "bad block timestamp",
		header: bc.BlockHeader{
			PreviousBlockHash:      prevHash,
			TransactionsMerkleRoot: emptyMerkleRoot,
			Height:                 2,
			TimestampMS:            3,
			Witness:                [][]byte{{0x04}},
		},
		want: ErrBadTimestamp,
	}, {
		desc: "fake initial block",
		header: bc.BlockHeader{
			PreviousBlockHash:      prevHash,
			TransactionsMerkleRoot: emptyMerkleRoot,
			Height:                 1,
			TimestampMS:            6,
			ConsensusProgram:       []byte{byte(vm.OP_5), byte(vm.OP_ADD), byte(vm.OP_9), byte(vm.OP_EQUAL)},
			Witness:                [][]byte{{0x04}},
		},
		want: ErrBadHeight,
	}, {
		desc: "bad block output script",
		header: bc.BlockHeader{
			PreviousBlockHash:      prevHash,
			TransactionsMerkleRoot: emptyMerkleRoot,
			Height:                 2,
			TimestampMS:            6,
			ConsensusProgram:       []byte{byte(vm.OP_FAIL)},
			Witness:                [][]byte{{0x04}},
		},
		want: ErrBadScript,
	}, {
		desc: "bad block signature script",
		header: bc.BlockHeader{
			PreviousBlockHash:      prevHash,
			TransactionsMerkleRoot: emptyMerkleRoot,
			Height:                 2,
			TimestampMS:            6,
			ConsensusProgram:       []byte{byte(vm.OP_5), byte(vm.OP_ADD), byte(vm.OP_9), byte(vm.OP_EQUAL)},
			Witness:                [][]byte{{0x03}},
		},
		want: ErrBadSig,
	}, {
		desc: "valid header",
		header: bc.BlockHeader{
			PreviousBlockHash:      prevHash,
			TransactionsMerkleRoot: emptyMerkleRoot,
			Height:                 2,
			TimestampMS:            6,
			ConsensusProgram:       []byte{byte(vm.OP_5), byte(vm.OP_ADD), byte(vm.OP_9), byte(vm.OP_EQUAL)},
			Witness:                [][]byte{{0x04}},
		},
		want: nil,
	}}
	for i, c := range cases {
		block := &bc.Block{BlockHeader: c.header}
		snap := state.Empty()
		got := ValidateBlockForAccept(ctx, snap, prevHash, prev, block, nil) // nil b/c no txs to validate
		if errors.Root(got) != c.want {
			t.Errorf("%d", i)
			t.Errorf("%s: got %q want %q", c.desc, got, c.want)
		}
	}
}
