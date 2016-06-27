package validation

import (
	"bytes"
	"chain/cos/bc"
	"testing"
)

func TestCalcMerkleRoot(t *testing.T) {
	cases := []struct {
		hashes []bc.Hash
		want   bc.Hash
	}{{
		hashes: []bc.Hash{
			mustParseHash("5df6e0e2761359d30a8275058e299fcc0381534545f55cf43e41983f5d4c9456"),
		},
		want: mustParseHash("5df6e0e2761359d30a8275058e299fcc0381534545f55cf43e41983f5d4c9456"),
	}, {
		hashes: []bc.Hash{
			mustParseHash("9c2e4d8fe97d881430de4e754b4205b9c27ce96715231cffc4337340cb110280"),
			mustParseHash("0c08173828583fc6ecd6ecdbcca7b6939c49c242ad5107e39deb7b0a5996b903"),
		},
		want: mustParseHash("3209f27d2c5800f7c6efd2b488498624d18896c4796cf0a5721b0b918e8b6c5b"),
	}, {
		hashes: []bc.Hash{
			mustParseHash("9c2e4d8fe97d881430de4e754b4205b9c27ce96715231cffc4337340cb110280"),
			mustParseHash("0c08173828583fc6ecd6ecdbcca7b6939c49c242ad5107e39deb7b0a5996b903"),
			mustParseHash("80903da4e6bbdf96e8ff6fc3966b0cfd355c7e860bdd1caa8e4722d9230e40ac"),
		},
		want: mustParseHash("c4ae6d8297d908b4f1acc68ee8ed73d64e925f2bbd2494400592ddc2319dda7e"),
	}}

	for _, c := range cases {
		var txs []*bc.Tx
		for _, h := range c.hashes {
			txs = append(txs, &bc.Tx{Hash: h})
		}
		got := CalcMerkleRoot(txs)

		if !bytes.Equal(got[:], c.want[:]) {
			t.Log("hashes", c.hashes)
			t.Errorf("got merkle root = %s want %s", got, c.want)
		}
	}
}

func mustParseHash(s string) bc.Hash {
	h, err := bc.ParseHash(s)
	if err != nil {
		panic(err)
	}
	return h
}
