package validation

import (
	"testing"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/errors"
)

func TestValidateBlockHeader(t *testing.T) {
	ctx := context.Background()
	prevBlock := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Height:       1,
			Timestamp:    5,
			OutputScript: []byte{txscript.OP_5, txscript.OP_ADD, txscript.OP_9, txscript.OP_EQUAL},
		},
	}
	prevHash := prevBlock.Hash()
	cases := []struct {
		desc   string
		header bc.BlockHeader
		want   error
	}{{
		desc: "bad prev block hash",
		header: bc.BlockHeader{
			PreviousBlockHash: bc.Hash{},
			Height:            2,
		},
		want: ErrBadPrevHash,
	}, {
		desc: "bad block height",
		header: bc.BlockHeader{
			PreviousBlockHash: prevHash,
			Height:            3,
		},
		want: ErrBadHeight,
	}, {
		desc: "bad block timestamp",
		header: bc.BlockHeader{
			PreviousBlockHash: prevHash,
			Height:            2,
			Timestamp:         3,
		},
		want: ErrBadTimestamp,
	}, {
		desc: "fake genesis block",
		header: bc.BlockHeader{
			PreviousBlockHash: prevHash,
			Height:            1,
			Timestamp:         6,
			OutputScript:      []byte{txscript.OP_5, txscript.OP_ADD, txscript.OP_9, txscript.OP_EQUAL},
			SignatureScript:   []byte{txscript.OP_4},
		},
		want: ErrBadHeight,
	}, {
		desc: "bad block output script",
		header: bc.BlockHeader{
			PreviousBlockHash: prevHash,
			Height:            2,
			Timestamp:         6,
			OutputScript:      []byte{txscript.OP_RETURN},
		},
		want: ErrBadScript,
	}, {
		desc: "bad block signature script",
		header: bc.BlockHeader{
			PreviousBlockHash: prevHash,
			Height:            2,
			Timestamp:         6,
			OutputScript:      []byte{txscript.OP_5, txscript.OP_ADD, txscript.OP_9, txscript.OP_EQUAL},
			SignatureScript:   []byte{txscript.OP_3},
		},
		want: ErrBadSig,
	}, {
		desc: "valid header",
		header: bc.BlockHeader{
			PreviousBlockHash: prevHash,
			Height:            2,
			Timestamp:         6,
			OutputScript:      []byte{txscript.OP_5, txscript.OP_ADD, txscript.OP_9, txscript.OP_EQUAL},
			SignatureScript:   []byte{txscript.OP_4},
		},
		want: nil,
	}}
	for _, c := range cases {
		block := &bc.Block{BlockHeader: c.header}
		got := ValidateBlockHeader(ctx, prevBlock, block)

		if errors.Root(got) != c.want {
			t.Errorf("%s: got %q want %q", c.desc, got, c.want)
		}
	}
}
