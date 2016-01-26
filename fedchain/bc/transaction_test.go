package bc

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/txscript"

	"chain/errors"
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
		tx.writeTo(ioutil.Discard, true)
	}
}

func BenchmarkTxWriteToFalse(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, false)
	}
}

func BenchmarkTxWriteToTrue200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, &TxInput{})
		tx.Outputs = append(tx.Outputs, &TxOutput{})
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, true)
	}
}

func BenchmarkTxWriteToFalse200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, &TxInput{})
		tx.Outputs = append(tx.Outputs, &TxOutput{})
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, false)
	}
}

func BenchmarkTxInputWriteToTrue(b *testing.B) {
	input := &TxInput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew, true)
	}
}

func BenchmarkTxInputWriteToFalse(b *testing.B) {
	input := &TxInput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew, false)
	}
}

func BenchmarkTxOutputWriteToTrue(b *testing.B) {
	output := &TxOutput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew, true)
	}
}

func BenchmarkTxOutputWriteToFalse(b *testing.B) {
	output := &TxOutput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew, false)
	}
}
