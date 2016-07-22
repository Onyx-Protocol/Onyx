package validation

import (
	"bytes"
	"chain/cos/bc"
	"chain/cos/txscript"
	"testing"
)

func TestCalcMerkleRoot(t *testing.T) {
	cases := []struct {
		witnesses [][][]byte
		want      bc.Hash
	}{{
		witnesses: [][][]byte{
			[][]byte{
				txscript.NumItem(1).Bytes(),
				[]byte("00000"),
			},
		},
		want: mustParseHash("108bfcf00f2d5a5b4d0dee0c4292e4175aec4c84a858b3e09dfcdf7fa8ab44a5"),
	}, {
		witnesses: [][][]byte{
			[][]byte{
				txscript.NumItem(1).Bytes(),
				[]byte("000000"),
			},
			[][]byte{
				txscript.NumItem(1).Bytes(),
				[]byte("111111"),
			},
		},
		want: mustParseHash("143a6b821416aae42b8ea24a92530dff4fc0b94d19db134870a8cd1007c316ea"),
	}, {
		witnesses: [][][]byte{
			[][]byte{
				txscript.NumItem(1).Bytes(),
				[]byte("000000"),
			},
			[][]byte{
				txscript.NumItem(2).Bytes(),
				[]byte("111111"),
				[]byte("222222"),
			},
		},
		want: mustParseHash("1e625cf93dde38aafa0232c41229e01ffe2083a4612cf3679b5e1149e88c62b4"),
	}}

	for _, c := range cases {
		var txs []*bc.Tx
		for _, wit := range c.witnesses {
			txs = append(txs, &bc.Tx{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						&bc.TxInput{
							AssetVersion:    1,
							InputCommitment: &bc.SpendInputCommitment{},
							InputWitness:    wit,
						},
					},
				},
			})
		}
		got := CalcMerkleRoot(txs)
		if !bytes.Equal(got[:], c.want[:]) {
			t.Log("witnesses", c.witnesses)
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
