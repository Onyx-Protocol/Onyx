package validation

import (
	"bytes"
	"testing"

	"chain/protocol/bc"
	"chain/protocol/vm"
)

func TestCalcMerkleRoot(t *testing.T) {
	cases := []struct {
		witnesses [][][]byte
		want      bc.Hash
	}{{
		witnesses: [][][]byte{
			[][]byte{
				vm.Int64Bytes(1),
				[]byte("00000"),
			},
		},
		want: mustParseHash("cee4bfb7d5c56f1c23258010e8a9b44278a1103c21e9600937e620bceca756f2"),
	}, {
		witnesses: [][][]byte{
			[][]byte{
				vm.Int64Bytes(1),
				[]byte("000000"),
			},
			[][]byte{
				vm.Int64Bytes(1),
				[]byte("111111"),
			},
		},
		want: mustParseHash("04cfb2705d678e0a33f1cf476b75f301d4a6dcc8ac33f0b9b43298f6527bf3f2"),
	}, {
		witnesses: [][][]byte{
			[][]byte{
				vm.Int64Bytes(1),
				[]byte("000000"),
			},
			[][]byte{
				vm.Int64Bytes(2),
				[]byte("111111"),
				[]byte("222222"),
			},
		},
		want: mustParseHash("a6042b3f195ad2c938b198fcd8346ff266814a464bb8b40f9e846160abf9be02"),
	}}

	for _, c := range cases {
		var txs []*bc.Tx
		for _, wit := range c.witnesses {
			txs = append(txs, &bc.Tx{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						&bc.TxInput{
							AssetVersion: 1,
							TypedInput: &bc.SpendInput{
								Arguments: wit,
							},
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
