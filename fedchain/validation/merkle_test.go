package validation

import (
	"bytes"
	"chain/fedchain/bc"
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
		want: mustParseHash("7de236613dd3d9fa1d86054a84952f1e0df2f130546b394a4d4dd7b76997f607"),
	}, {
		hashes: []bc.Hash{
			mustParseHash("9c2e4d8fe97d881430de4e754b4205b9c27ce96715231cffc4337340cb110280"),
			mustParseHash("0c08173828583fc6ecd6ecdbcca7b6939c49c242ad5107e39deb7b0a5996b903"),
			mustParseHash("80903da4e6bbdf96e8ff6fc3966b0cfd355c7e860bdd1caa8e4722d9230e40ac"),
		},
		want: mustParseHash("5b7534123197114fa7e7459075f39d89ffab74b5c3f31fad48a025b931ff5a01"),
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
