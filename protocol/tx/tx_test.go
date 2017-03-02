package tx

import (
	"encoding/hex"
	"testing"

	"chain/protocol/bc"
)

func TestTxHashes(t *testing.T) {
	cases := []struct {
		txdata *bc.TxData
		hash   bc.Hash
	}{
		{
			txdata: &bc.TxData{},
			hash:   mustDecodeHash("e367a95b0f1dafdedd86f633456c81ef6bd4f2623f0890d56417f73a18a67297"),
		},
		{
			txdata: sampleTx(),
			hash:   mustDecodeHash("9fad4f5024412d99d17508ef3cc66f81f1e09914a71b2641683acca87081c098"), // todo: verify this value,
		},
	}

	for i, c := range cases {
		hashes, err := TxHashes(c.txdata)
		if err != nil {
			t.Fatal(err)
		}
		if len(hashes.VMContexts) != len(c.txdata.Inputs) {
			t.Errorf("case %d: len(hashes.VMContexts) = %d, want %d", i, len(hashes.VMContexts), len(c.txdata.Inputs))
		}
		if c.hash != hashes.ID {
			t.Errorf("case %d: got txid %x, want %x", i, hashes.ID[:], c.hash[:])
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
			bc.NewSpendInput(nil, mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), assetID, 1000000000000, 1, []byte{1}, bc.Hash{}, []byte("input")),
			bc.NewSpendInput(nil, bc.Hash{17}, assetID, 1, 1, []byte{2}, bc.Hash{}, []byte("input2")),
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
