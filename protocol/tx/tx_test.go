package tx

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/protocol/bc"
)

func TestMapTx(t *testing.T) {
	// sample data copied from protocol/bc/transaction_test.go

	issuanceScript := []byte{1}
	initialBlockHashHex := "03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"
	initialBlockHash := mustDecodeHash(initialBlockHashHex)

	oldTx := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(bc.ComputeOutputID(mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), 0), nil, bc.AssetID{}, 1000000000000, []byte{1}, []byte("input")),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(bc.ComputeAssetID(issuanceScript, initialBlockHash, 1, bc.EmptyStringHash), 600000000000, []byte{1}, nil),
			bc.NewTxOutput(bc.ComputeAssetID(issuanceScript, initialBlockHash, 1, bc.EmptyStringHash), 400000000000, []byte{2}, nil),
		},
		MinTime:       1492590000,
		MaxTime:       1492590591,
		ReferenceData: []byte("distribution"),
	}

	_, entryMap, err := mapTx(oldTx)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(spew.Sdump(entryMap))
}

func mustDecodeHash(hash string) (h [32]byte) {
	if len(hash) != hex.EncodedLen(len(h)) {
		panic("wrong length hash")
	}
	_, err := hex.Decode(h[:], []byte(hash))
	if err != nil {
		panic(err)
	}
	return h
}
