package bc_test

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"reflect"
	"testing"

	"chain/errors"
	. "chain/fedchain/bc"
	"chain/fedchain/txscript"
)

func TestTransaction(t *testing.T) {
	issuanceScript := []byte{txscript.OP_1}
	genesisHash := mustDecodeHash("03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d")

	cases := []struct {
		tx          *Tx
		hex         string
		hash        [32]byte
		witnessHash [32]byte
	}{
		{
			tx: NewTx(TxData{
				Version:  1,
				Inputs:   nil,
				Outputs:  nil,
				LockTime: 0,
				Metadata: nil,
			}),
			hex:         "010000000000000000000000000000",
			hash:        mustDecodeHash("d64277a66bbd1a66e12ee31797f7b9d2487e056def294e5f5240e64e0324ad45"),
			witnessHash: mustDecodeHash("bb0e9f24579bab40b88df4b409984ef7fdcb1a9416ba5d89e6009f6f7358214d"),
		},
		{
			tx: NewTx(TxData{
				Version: 1,
				Inputs: []*TxInput{
					{
						Previous: Outpoint{
							Hash:  mustDecodeHash("03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"),
							Index: InvalidOutputIndex,
						},
						// "PUSHDATA 'issuance'"
						SignatureScript: []byte{txscript.OP_DATA_8, 0x69, 0x73, 0x73, 0x75, 0x61, 0x6e, 0x63, 0x65},
						Metadata:        []byte("input"),
					},
				},
				Outputs: []*TxOutput{
					{
						AssetAmount: AssetAmount{AssetID: AssetID{}, Amount: 1000000000000},
						Script:      []byte{txscript.OP_1},
						Metadata:    []byte("output"),
					},
				},
				LockTime: 0,
				Metadata: []byte("issuance"),
			}),
			hex:         "010000000103deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758dffffffff090869737375616e636505696e707574000100000000000000000000000000000000000000000000000000000000000000000010a5d4e80000000151066f757470757400000000000000000869737375616e6365",
			hash:        mustDecodeHash("fa104295fcf2dc017cab5ce66b1306e070478311dde94cf4ad5c874934ffbfcf"),
			witnessHash: mustDecodeHash("f0c903c599c5b963c51d63cfd7025e0f5c6d368f95cfa4a684ff6bf49325ebab"),
		},
		{
			tx: NewTx(TxData{
				Version: 1,
				Inputs: []*TxInput{
					{
						Previous: Outpoint{
							Hash:  mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"),
							Index: 0,
						},
						SignatureScript: nil,
						Metadata:        []byte("input"),
						AssetDefinition: []byte("assetdef"),
					},
				},
				Outputs: []*TxOutput{
					{
						AssetAmount: AssetAmount{AssetID: ComputeAssetID(issuanceScript, genesisHash), Amount: 600000000000},
						Script:      []byte{txscript.OP_1},
						Metadata:    nil,
					},
					{
						AssetAmount: AssetAmount{AssetID: ComputeAssetID(issuanceScript, genesisHash), Amount: 400000000000},
						Script:      []byte{txscript.OP_2},
						Metadata:    nil,
					},
				},
				LockTime: 1492590591,
				Metadata: []byte("distribution"),
			}),
			hex:         "0100000001dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292000000000005696e7075740861737365746465660265dad3e60971527b1158344272b23ef9634cc72ce69ce54d501b5293dce0ef7c0070c9b28b00000001510065dad3e60971527b1158344272b23ef9634cc72ce69ce54d501b5293dce0ef7c00a0db215d000000015200ff1ff758000000000c646973747269627574696f6e",
			hash:        mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151"),
			witnessHash: mustDecodeHash("c0c7aae5cca172a06265670cabce02d4f3efbbafdb8e4e8e60f84b1822bb64d9"),
		},
	}

	for _, test := range cases {
		t.Logf("metadata %q", test.tx.Metadata)

		got := serialize(t, test.tx)
		want, _ := hex.DecodeString(test.hex)
		if !bytes.Equal(got, want) {
			t.Errorf("bytes = %x want %x", got, want)
		}
		if test.tx.Hash != test.hash {
			t.Errorf("hash = %s want %x", test.tx.Hash, test.hash)
		}
		if g := test.tx.WitnessHash(); g != test.witnessHash {
			t.Errorf("witness hash = %s want %x", g, test.witnessHash)
		}

		tx1 := new(TxData)
		err := tx1.UnmarshalText([]byte(test.hex))
		if err != nil {
			t.Errorf("unexpected err %v", err)
		}
		if !reflect.DeepEqual(*tx1, test.tx.TxData) {
			t.Errorf("tx1 = %v want %v", *tx1, test.tx.TxData)
		}
	}
}

func TestIsIssuance(t *testing.T) {
	tx := NewTx(TxData{
		Version: CurrentTransactionVersion,
		Inputs: []*TxInput{
			{
				Previous: Outpoint{
					Hash:  mustDecodeHash("dd506f5d4c3f904d3d4b3c3be597c9198c6193ffd14a28570e4a923ce40cf9e5"),
					Index: InvalidOutputIndex,
				},
				// "PUSHDATA 'issuance'"
				SignatureScript: []byte{txscript.OP_DATA_8, 0x69, 0x73, 0x73, 0x75, 0x61, 0x6e, 0x63, 0x65},
				Metadata:        []byte("input"),
			},
		},
		Outputs: []*TxOutput{
			{
				AssetAmount: AssetAmount{AssetID: AssetID{}, Amount: 1000000000000},
				Script:      []byte{txscript.OP_1},
				Metadata:    []byte("output"),
			},
		},
		LockTime: 0,
		Metadata: []byte("issuance"),
	})

	if g := tx.Inputs[0].IsIssuance(); !g {
		t.Errorf("input IsIssuance() = %v want true", g)
	}
	if g := tx.IsIssuance(); !g {
		t.Errorf("tx IsIssuance() = %v want true", g)
	}
}

func TestEmptyOutpoint(t *testing.T) {
	g := Outpoint{}.String()
	w := "0000000000000000000000000000000000000000000000000000000000000000:0"
	if g != w {
		t.Errorf("Empty outpoint has incorrect string representation '%v'", g)
	}
}

func TestIssuanceOutpoint(t *testing.T) {
	hex := "fbc27d22c48b9b2533c4e97f7863f3dca805b8924ea2b7c6783f3fd99fdb2c29"
	o := Outpoint{
		Hash:  mustDecodeHash(hex),
		Index: 0xffffffff,
	}
	if got := o.String(); got != hex+":4294967295" {
		t.Errorf("Issuance outpoint has incorrect string representation '%v'", got)
	}
}

func TestOutpointWriteErr(t *testing.T) {
	var w errWriter
	var p Outpoint
	_, err := p.WriteTo(&w)
	if err == nil {
		t.Error("outpoint WriteTo(w) err = nil; want non-nil error")
	}
}

func TestTxHashForSig(t *testing.T) {
	assetID := ComputeAssetID([]byte{txscript.OP_1}, mustDecodeHash("03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"))
	tx := &TxData{
		Version: 1,
		Inputs: []*TxInput{{
			Previous:        Outpoint{Hash: mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151")},
			SignatureScript: []byte{txscript.OP_1},
			Metadata:        []byte("input1"),
		}, {
			Previous:        Outpoint{Hash: mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151"), Index: 1},
			SignatureScript: []byte{txscript.OP_2},
		}},
		Outputs: []*TxOutput{{
			AssetAmount: AssetAmount{AssetID: assetID, Amount: 1000000000000},
			Script:      []byte{txscript.OP_3},
		}},
		Metadata: []byte("transfer"),
	}
	cases := []struct {
		idx      int
		hashType SigHashType
		wantHash string
	}{
		{0, SigHashAll, "c0c9a7e21ae86ad8e00080193a0eeacda58bef2f20b1a4f26a25f9276a52a6a6"},
		{0, SigHashSingle, "ea8736b45276e1ad35933c616c6c1b064dae85a2f254fd4c599ea983feb0b8ae"},
		{0, SigHashNone, "e1f8f931e6cad3a8220dff0f311f6776e0bec1c6ef2942cfc4214ffcee394b1b"},
		{0, SigHashAll | SigHashAnyOneCanPay, "87d5decac7242727351ed312a8fc9a72452e371a086f4c4007cbe5d95f681481"},
		{0, SigHashSingle | SigHashAnyOneCanPay, "18648d6c5ecd58ddd255c69439b461420abc7afc0c7d392b24a25abe3aca5a8a"},
		{0, SigHashNone | SigHashAnyOneCanPay, "36029bf2b9d5253c6554644e537eedf38791341ee3ef36cc1bcce13bd6cd451c"},

		{1, SigHashAll, "2859efe72e7b4dc7e010feb896a489a19220c8b784c1da1505217a017c9913f5"},
		{1, SigHashSingle, "4e1caa039e66bd99eae2309d90d612407cca585d685e2e1232144473918c3365"},
		{1, SigHashNone, "19b604471a3566aea416116dd3048d8195214566230e0007341b28a66cad838b"},
		{1, SigHashAll | SigHashAnyOneCanPay, "a6afa32a2e5d47cccda3c75383e1b8338b7dde8310b4beed39e3d4d9ba4a43e0"},
		{1, SigHashSingle | SigHashAnyOneCanPay, "4c4f465a026b8ea06e73c34b37d83cabadb3d34b4dc468dc8e237f10fecab1df"},
		{1, SigHashNone | SigHashAnyOneCanPay, "1acc09ba0ba78006c024a88f90e0fce8997db7c32b25709873df65907a88352f"},
	}
	assetAmount := tx.Outputs[0].AssetAmount
	cache := &SigHashCache{}

	for _, c := range cases {
		hash := tx.HashForSig(c.idx, assetAmount, c.hashType)

		if hash.String() != c.wantHash {
			t.Errorf("HashForSig(%d, %v) = %s want %s", c.idx, c.hashType, hash.String(), c.wantHash)
		}

		cachedHash := tx.HashForSigCached(c.idx, assetAmount, c.hashType, cache)

		if cachedHash.String() != c.wantHash {
			t.Errorf("HashForSigCached(%d, %v) = %s want %s", c.idx, c.hashType, hash.String(), c.wantHash)
		}
	}
}

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) {
	return 0, errors.New("bad write")
}

func BenchmarkTxHash(b *testing.B) {
	tx := &TxData{}
	for i := 0; i < b.N; i++ {
		tx.Hash()
	}
}

func BenchmarkTxWriteToTrue(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < b.N; i++ {
		tx.WriteToForHash(ioutil.Discard, true)
	}
}

func BenchmarkTxWriteToFalse(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < b.N; i++ {
		tx.WriteToForHash(ioutil.Discard, false)
	}
}

func BenchmarkTxWriteToTrue200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, &TxInput{})
		tx.Outputs = append(tx.Outputs, &TxOutput{})
	}
	for i := 0; i < b.N; i++ {
		tx.WriteToForHash(ioutil.Discard, true)
	}
}

func BenchmarkTxWriteToFalse200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, &TxInput{})
		tx.Outputs = append(tx.Outputs, &TxOutput{})
	}
	for i := 0; i < b.N; i++ {
		tx.WriteToForHash(ioutil.Discard, false)
	}
}

func BenchmarkTxInputWriteToTrue(b *testing.B) {
	input := &TxInput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.WriteTo(ew, true)
	}
}

func BenchmarkTxInputWriteToFalse(b *testing.B) {
	input := &TxInput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.WriteTo(ew, false)
	}
}

func BenchmarkTxOutputWriteToTrue(b *testing.B) {
	output := &TxOutput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.WriteTo(ew, true)
	}
}

func BenchmarkTxOutputWriteToFalse(b *testing.B) {
	output := &TxOutput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.WriteTo(ew, false)
	}
}
