package validation

import (
	"testing"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
)

// emptyMerkleRoot is the SHA3-256 of "".
var emptyMerkleRoot = mustParseHash("a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a")

func TestValidateBlockHeader(t *testing.T) {
	prevHeader := bc.BlockHeader{
		Height:           1,
		TimestampMS:      5,
		ConsensusProgram: []byte{byte(vm.OP_5), byte(vm.OP_ADD), byte(vm.OP_9), byte(vm.OP_EQUAL)},
	}
	prevHash := prevHeader.Hash()
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
		},
		want: ErrBadPrevHash,
	}, {
		desc: "bad block height",
		header: bc.BlockHeader{
			PreviousBlockHash:      prevHash,
			TransactionsMerkleRoot: emptyMerkleRoot,
			Height:                 3,
		},
		want: ErrBadHeight,
	}, {
		desc: "bad block timestamp",
		header: bc.BlockHeader{
			PreviousBlockHash:      prevHash,
			TransactionsMerkleRoot: emptyMerkleRoot,
			Height:                 2,
			TimestampMS:            3,
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
		got := validateBlockHeader(&prevHeader, block, true)
		if errors.Root(got) != c.want {
			t.Errorf("%d", i)
			t.Errorf("%s: got %q want %q", c.desc, got, c.want)
		}
	}
}
