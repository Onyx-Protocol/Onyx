package validation

import (
	"bytes"
	"chain/cos/bc"
	"chain/cos/vm"
	"testing"
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
		want: mustParseHash("b2e80b0f134f1035816c1bc3d37b962b8f92be9097d8ba1fba2dc543e9e54231"),
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
		want: mustParseHash("d58a9f8db787fa316865ef1cca70e9e3aeb8bc1c9c7a2bccb6e1b4d2bdc2b270"),
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
		want: mustParseHash("9acb338f300707b0fd06e8f0b6c978fadc4bcf3efda98eb2664d63c7057b8c14"),
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
