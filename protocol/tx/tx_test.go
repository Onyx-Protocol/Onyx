package tx

import (
	"encoding/hex"
	"testing"

	"chain/protocol/bc"
)

func TestTxHashes(t *testing.T) {
	cases := []*bc.TxData{{}, sampleTx()}

	for _, txData := range cases {
		hashes, err := TxHashes(txData)
		if err != nil {
			t.Fatal(err)
		}
		if len(hashes.VMContexts) != len(txData.Inputs) {
			t.Errorf("len(hashes.VMContexts) = %d, want %d", len(hashes.VMContexts), len(txData.Inputs))
		}
	}
}

func BenchmarkHashEmptyTx(b *testing.B) {
	tx := &bc.TxData{}
	for i := 0; i < b.N; i++ {
		_, err := TxHashes(tx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashNonemptyTx(b *testing.B) {
	tx := sampleTx()
	for i := 0; i < b.N; i++ {
		_, err := TxHashes(tx)
		if err != nil {
			b.Fatal(err)
		}
	}
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

func sampleTx() *bc.TxData {
	assetID := bc.ComputeAssetID([]byte{1}, mustDecodeHash("03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"), 1, bc.EmptyStringHash)
	return &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
<<<<<<< e69df8246b33bf70f67c8e8586e5e5d62ec94666
			bc.NewSpendInput(bc.ComputeOutputID(mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), 0), nil, assetID, 1000000000000, []byte{1}, []byte("input")),
=======
			bc.NewSpendInput(bc.OutputID{mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292")}, nil, assetID, 1000000000000, []byte{1}, []byte("input")),
>>>>>>> wip: begin txgraph outline
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 600000000000, []byte{1}, nil),
			bc.NewTxOutput(assetID, 400000000000, []byte{2}, nil),
		},
		MinTime:       1492590000,
		MaxTime:       1492590591,
		ReferenceData: []byte("distribution"),
	}
}
