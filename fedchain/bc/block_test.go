package bc

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	//	"github.com/btcsuite/btcd/txscript"
)

func TestEmptyBlock(t *testing.T) {
	block := Block{
		BlockHeader: BlockHeader{
			Version: NewBlockVersion,
		},
	}

	got := serialize(t, &block)
	want, _ := hex.DecodeString("0000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	if !bytes.Equal(got, want) {
		t.Errorf("empty block bytes = %x want %x", got, want)
	}

	got = serialize(t, &block.BlockHeader)
	want, _ = hex.DecodeString("00000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	if !bytes.Equal(got, want) {
		t.Errorf("empty block header bytes = %x want %x", got, want)
	}

	wantHash := mustDecodeHash("a78f912c5c47f4377f97be66ba1551fcaa9437396296931e1b095013e97abb4e")
	if h := block.Hash(); !bytes.Equal(h[:], wantHash[:]) {
		t.Errorf("empty block has incorrect hash %x", h)
	}
	if h := block.HashForSig(); !bytes.Equal(h[:], wantHash[:]) {
		t.Errorf("empty block has incorrect sig hash %x", h)
	}

	wTime := time.Unix(0, 0).UTC()
	if got := block.Time(); got != wTime {
		t.Errorf("empty block time = %v want %v", got, wTime)
	}
}

func TestSmallBlock(t *testing.T) {
	block := Block{
		BlockHeader: BlockHeader{
			Version: NewBlockVersion,
		},
		Transactions: []*Tx{{Version: CurrentTransactionVersion}},
	}

	got := serialize(t, &block)
	want, _ := hex.DecodeString("0000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000010000000000000000000000")
	if !bytes.Equal(got, want) {
		t.Errorf("small block bytes = %x want %x", got, want)
	}
}
