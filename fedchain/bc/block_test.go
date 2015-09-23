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
	want, _ := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	if !bytes.Equal(got, want) {
		t.Errorf("empty block bytes = %x want %x", got, want)
	}

	got = serialize(t, &block.BlockHeader)
	want, _ = hex.DecodeString("01000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	if !bytes.Equal(got, want) {
		t.Errorf("empty block header bytes = %x want %x", got, want)
	}

	h := block.Hash()
	if id := ID(h[:]); id != "766d9541811800a40f2b976fcbb508178e31704167b87f5d8babf8ff951c07af" {
		t.Errorf("empty block has incorrect ID %s", id)
	}

	h = block.HashForSig()
	if got := hex.EncodeToString(h[:]); got != "af071c95fff8ab8b5d7fb8674170318e1708b5cb6f972b0fa400188141956d76" {
		t.Errorf("empty block has incorrect ID %s", got)
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
		Transactions: []Tx{{Version: CurrentTransactionVersion}},
	}

	got := serialize(t, &block)
	want, _ := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001010000000000000000000000000000")
	if !bytes.Equal(got, want) {
		t.Errorf("small block bytes = %x want %x", got, want)
	}
}
