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
		want: mustParseHash("394b0904ac32c1818cb81c0a21e0b4ffcd2526d47b40349c554c4c588ea6a791"),
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
		want: mustParseHash("11ba8c6ad7e62dcdcd792467ec234365f19f085bc4d323d42654e150e88c81d6"),
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
		want: mustParseHash("dfe61cf68e70ad29b2bd52b734730dc3a99e7681c74fc17554a1e4bc0a9c80ae"),
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
