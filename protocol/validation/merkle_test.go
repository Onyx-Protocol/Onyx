package validation

import (
	"bytes"
	"testing"
	"time"

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
		want: mustParseHash("c9678d813c3ed1d5b8f31d1fa95471d49cd34a8d8d25ceb879b6f1fa718b69c0"),
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
		want: mustParseHash("0cd10ea2f25a217027529d83dae0ce9c5f308490e2eaebb5bf234c6795cdbd06"),
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
		want: mustParseHash("e2b16f65a2c02bfda54ff9929011f0931914a1f9f4e5c3af0033ef0d4c98b89d"),
	}}

	for _, c := range cases {
		var txs []*bc.Tx
		for _, wit := range c.witnesses {
			txs = append(txs, bc.NewTx(bc.TxData{
				Inputs: []*bc.TxInput{
					&bc.TxInput{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							Arguments: wit,
						},
					},
				},
			}))
		}
		got, err := CalcMerkleRoot(txs)
		if err != nil {
			t.Fatalf("unexpected error %s", err)
		}
		if !bytes.Equal(got[:], c.want[:]) {
			t.Log("witnesses", c.witnesses)
			t.Errorf("got merkle root = %s want %s", got, c.want)
		}
	}
}

func TestDuplicateLeaves(t *testing.T) {
	var initialBlockHash bc.Hash
	trueProg := []byte{byte(vm.OP_TRUE)}
	assetID := bc.ComputeAssetID(trueProg, initialBlockHash, 1, bc.EmptyStringHash)
	txs := make([]*bc.Tx, 6)
	for i := uint64(0); i < 6; i++ {
		now := []byte(time.Now().String())
		txs[i] = bc.NewTx(bc.TxData{
			Version: 1,
			Inputs:  []*bc.TxInput{bc.NewIssuanceInput(now, i, nil, initialBlockHash, trueProg, nil, nil)},
			Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, i, trueProg, nil)},
		})
	}

	// first, get the root of an unbalanced tree
	txns := []*bc.Tx{txs[5], txs[4], txs[3], txs[2], txs[1], txs[0]}
	root1, err := CalcMerkleRoot(txns)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	// now, get the root of a balanced tree that repeats leaves 0 and 1
	txns = []*bc.Tx{txs[5], txs[4], txs[3], txs[2], txs[1], txs[0], txs[1], txs[0]}
	root2, err := CalcMerkleRoot(txns)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	if root1 == root2 {
		t.Error("forged merkle tree by duplicating some leaves")
	}
}

func TestAllDuplicateLeaves(t *testing.T) {
	var initialBlockHash bc.Hash
	trueProg := []byte{byte(vm.OP_TRUE)}
	assetID := bc.ComputeAssetID(trueProg, initialBlockHash, 1, bc.EmptyStringHash)
	now := []byte(time.Now().String())
	issuanceInp := bc.NewIssuanceInput(now, 1, nil, initialBlockHash, trueProg, nil, nil)

	tx := bc.NewTx(bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{issuanceInp},
		Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 1, trueProg, nil)},
	})
	tx1, tx2, tx3, tx4, tx5, tx6 := tx, tx, tx, tx, tx, tx

	// first, get the root of an unbalanced tree
	txs := []*bc.Tx{tx6, tx5, tx4, tx3, tx2, tx1}
	root1, err := CalcMerkleRoot(txs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	// now, get the root of a balanced tree that repeats leaves 5 and 6
	txs = []*bc.Tx{tx6, tx5, tx6, tx5, tx4, tx3, tx2, tx1}
	root2, err := CalcMerkleRoot(txs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	if root1 == root2 {
		t.Error("forged merkle tree with all duplicate leaves")
	}
}

func mustParseHash(s string) bc.Hash {
	h, err := bc.ParseHash(s)
	if err != nil {
		panic(err)
	}
	return h
}
