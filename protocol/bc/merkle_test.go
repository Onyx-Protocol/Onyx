package bc

import (
	"bytes"
	"testing"
	"time"

	"chain/protocol/vm"
)

func TestMerkleRoot(t *testing.T) {
	cases := []struct {
		witnesses [][][]byte
		want      Hash
	}{{
		witnesses: [][][]byte{
			[][]byte{
				{1},
				[]byte("00000"),
			},
		},
		want: mustParseHash("77eae4222f60bfd74c07994d700161d0b831ed723037952b9c7ee98ed8766977"),
	}, {
		witnesses: [][][]byte{
			[][]byte{
				{1},
				[]byte("000000"),
			},
			[][]byte{
				{1},
				[]byte("111111"),
			},
		},
		want: mustParseHash("526737fcca853f5ad352081c5a7341aca4ee05b09a002c8600e26a06df02aa3b"),
	}, {
		witnesses: [][][]byte{
			[][]byte{
				{1},
				[]byte("000000"),
			},
			[][]byte{
				{2},
				[]byte("111111"),
				[]byte("222222"),
			},
		},
		want: mustParseHash("526737fcca853f5ad352081c5a7341aca4ee05b09a002c8600e26a06df02aa3b"),
	}}

	for _, c := range cases {
		var txs []*TxEntries
		for _, wit := range c.witnesses {
			txs = append(txs, NewTx(TxData{
				Inputs: []*TxInput{
					&TxInput{
						AssetVersion: 1,
						TypedInput: &SpendInput{
							Arguments: wit,
						},
					},
				},
			}).TxEntries)
		}
		got, err := MerkleRoot(txs)
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
	var initialBlockHash Hash
	trueProg := []byte{byte(vm.OP_TRUE)}
	assetID := ComputeAssetID(trueProg, initialBlockHash, 1, EmptyStringHash)
	txs := make([]*TxEntries, 6)
	for i := uint64(0); i < 6; i++ {
		now := []byte(time.Now().String())
		txs[i] = NewTx(TxData{
			Version: 1,
			Inputs:  []*TxInput{NewIssuanceInput(now, i, nil, initialBlockHash, trueProg, nil, nil)},
			Outputs: []*TxOutput{NewTxOutput(assetID, i, trueProg, nil)},
		}).TxEntries
	}

	// first, get the root of an unbalanced tree
	txns := []*TxEntries{txs[5], txs[4], txs[3], txs[2], txs[1], txs[0]}
	root1, err := MerkleRoot(txns)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	// now, get the root of a balanced tree that repeats leaves 0 and 1
	txns = []*TxEntries{txs[5], txs[4], txs[3], txs[2], txs[1], txs[0], txs[1], txs[0]}
	root2, err := MerkleRoot(txns)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	if root1 == root2 {
		t.Error("forged merkle tree by duplicating some leaves")
	}
}

func TestAllDuplicateLeaves(t *testing.T) {
	var initialBlockHash Hash
	trueProg := []byte{byte(vm.OP_TRUE)}
	assetID := ComputeAssetID(trueProg, initialBlockHash, 1, EmptyStringHash)
	now := []byte(time.Now().String())
	issuanceInp := NewIssuanceInput(now, 1, nil, initialBlockHash, trueProg, nil, nil)

	tx := NewTx(TxData{
		Version: 1,
		Inputs:  []*TxInput{issuanceInp},
		Outputs: []*TxOutput{NewTxOutput(assetID, 1, trueProg, nil)},
	}).TxEntries
	tx1, tx2, tx3, tx4, tx5, tx6 := tx, tx, tx, tx, tx, tx

	// first, get the root of an unbalanced tree
	txs := []*TxEntries{tx6, tx5, tx4, tx3, tx2, tx1}
	root1, err := MerkleRoot(txs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	// now, get the root of a balanced tree that repeats leaves 5 and 6
	txs = []*TxEntries{tx6, tx5, tx6, tx5, tx4, tx3, tx2, tx1}
	root2, err := MerkleRoot(txs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	if root1 == root2 {
		t.Error("forged merkle tree with all duplicate leaves")
	}
}

func mustParseHash(s string) Hash {
	h, err := ParseHash(s)
	if err != nil {
		panic(err)
	}
	return h
}
