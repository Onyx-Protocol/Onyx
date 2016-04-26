package bc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"chain/errors"
)

func TestTransaction(t *testing.T) {
	issuanceScript := []byte{1}
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
						SignatureScript: []byte{1, 2, 3},
						Metadata:        []byte("input"),
					},
				},
				Outputs: []*TxOutput{
					{
						AssetAmount: AssetAmount{AssetID: AssetID{}, Amount: 1000000000000},
						Script:      []byte{1},
						Metadata:    []byte("output"),
					},
				},
				LockTime: 0,
				Metadata: []byte("issuance"),
			}),
			hex:         "010000000103deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758dffffffff00000000000000000000000000000000000000000000000000000000000000000000000000000000000301020305696e707574000100000000000000000000000000000000000000000000000000000000000000000010a5d4e80000000101066f757470757400000000000000000869737375616e6365",
			hash:        mustDecodeHash("cc8142e4584989fbbca4538ba49f75ce47d4b46dec3137a961fe25710dd3ee18"),
			witnessHash: mustDecodeHash("f3f4bcfd88a683c99dbdde750cb14022faf5db62549bd8a4f42a4e3f6e20da97"),
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
						AssetAmount:     AssetAmount{AssetID: AssetID{}, Amount: 1000000000000},
						PrevScript:      []byte{1},
						SignatureScript: nil,
						Metadata:        []byte("input"),
						AssetDefinition: []byte("assetdef"),
					},
				},
				Outputs: []*TxOutput{
					{
						AssetAmount: AssetAmount{AssetID: ComputeAssetID(issuanceScript, genesisHash), Amount: 600000000000},
						Script:      []byte{1},
						Metadata:    nil,
					},
					{
						AssetAmount: AssetAmount{AssetID: ComputeAssetID(issuanceScript, genesisHash), Amount: 400000000000},
						Script:      []byte{2},
						Metadata:    nil,
					},
				},
				LockTime: 1492590591,
				Metadata: []byte("distribution"),
			}),
			hex:         "0100000001dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d32920000000000000000000000000000000000000000000000000000000000000000000000000010a5d4e800000001010005696e707574086173736574646566028ff02bfb82be991185ea36426e233bb9d4b79797a669140b53ef21174fae43ad0070c9b28b0000000101008ff02bfb82be991185ea36426e233bb9d4b79797a669140b53ef21174fae43ad00a0db215d000000010200ff1ff758000000000c646973747269627574696f6e",
			hash:        mustDecodeHash("1ba1c708f98c24f0fa49eca066c4f9f4add27f0f3d14ccc1cfe4e26958bac0c5"),
			witnessHash: mustDecodeHash("30f68a7dd59556b0eea4c70ad29f8744ba8374f8ac600aaa43f32af6561ea737"),
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

		txJSON, err := json.Marshal(test.tx)
		if err != nil {
			t.Errorf("error marshaling tx to json: %s", err)
		}
		var txFromJSON Tx
		if err := json.Unmarshal(txJSON, &txFromJSON); err != nil {
			t.Errorf("error unmarshaling tx from json: %s", err)
		}
		if !reflect.DeepEqual(test.tx, &txFromJSON) {
			t.Errorf("bc.Tx -> json -> bc.Tx: got=%#v want=%#v", &txFromJSON, test.tx)
		}

		tx1 := new(TxData)
		if err := tx1.UnmarshalText([]byte(test.hex)); err != nil {
			t.Errorf("unexpected err %v", err)
		}
		if !reflect.DeepEqual(*tx1, test.tx.TxData) {
			t.Errorf("tx1 = %v want %v", *tx1, test.tx.TxData)
		}
	}
}

func TestHasIssuance(t *testing.T) {
	cases := []struct {
		tx   *TxData
		want bool
	}{{
		tx: &TxData{
			Inputs: []*TxInput{{Previous: Outpoint{Index: InvalidOutputIndex}}},
		},
		want: true,
	}, {
		tx: &TxData{
			Inputs: []*TxInput{{}, {Previous: Outpoint{Index: InvalidOutputIndex}}},
		},
		want: true,
	}, {
		tx: &TxData{
			Inputs: []*TxInput{{}},
		},
		want: false,
	}, {
		tx:   &TxData{},
		want: false,
	}}

	for _, c := range cases {
		got := c.tx.HasIssuance()
		if got != c.want {
			t.Errorf("HasIssuance(%+v) = %v want %v", c.tx, got, c.want)
		}
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
	ew := errors.NewWriter(w)
	p.WriteTo(ew)
	err := ew.Err()
	if err == nil {
		t.Error("outpoint WriteTo(w) err = nil; want non-nil error")
	}
}

func TestTxHashForSig(t *testing.T) {
	assetID := ComputeAssetID([]byte{1}, mustDecodeHash("03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"))
	tx := &TxData{
		Version: 1,
		Inputs: []*TxInput{{
			Previous:        Outpoint{Hash: mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151")},
			SignatureScript: []byte{1},
			Metadata:        []byte("input1"),
		}, {
			Previous:        Outpoint{Hash: mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151"), Index: 1},
			SignatureScript: []byte{2},
		}},
		Outputs: []*TxOutput{{
			AssetAmount: AssetAmount{AssetID: assetID, Amount: 1000000000000},
			Script:      []byte{3},
		}},
		Metadata: []byte("transfer"),
	}
	cases := []struct {
		idx      int
		hashType SigHashType
		wantHash string
	}{
		{0, SigHashAll, "0c7be972bdade58ee65d67572436ea1fc7d180f57d5a4facecdf9d00959fce0d"},
		{0, SigHashSingle, "fd10b4b679f36587adf80d1438ae909e8327f052854f7270fed58bb6b4ffc011"},
		{0, SigHashNone, "11417ff3e229e0f3c5a33906a3e7b2e32a3ec73a74e51255123026060818216e"},
		{0, SigHashAll | SigHashAnyOneCanPay, "d793e83a6124f30ab1a7b3311e25a78abeedb72e0e139d8bde7a60927793631d"},
		{0, SigHashSingle | SigHashAnyOneCanPay, "04c7bb5e754d96a896eb2f425b2458a0de2a6a1a77c28e1ac0015501c87cacee"},
		{0, SigHashNone | SigHashAnyOneCanPay, "852a61a9994cfb8249c3574ed16d14a22a92d90dfcec51d8d4284095ac29b421"},

		{1, SigHashAll, "078e85551c1c757c9dbd64fd9427d206a2512697bc9a9c7cc0264a6c83584f3e"},
		{1, SigHashSingle, "a5b1d4786ecf02eefd939e0308466517d8cc0c41ea811f616867c2d9a14ad577"},
		{1, SigHashNone, "232fb5c2557b993b528e477f1ef41c83e5a9e5eaa0942cd7c2395822b092823e"},
		{1, SigHashAll | SigHashAnyOneCanPay, "be91a5dc2b421cfecdd237a2c9c3f869e044c709bf3781c229839f8477bfdb02"},
		{1, SigHashSingle | SigHashAnyOneCanPay, "7a26e6428495349a7d017cd6f18dad1644438fba8172f88b48d21d9dfb977e74"},
		{1, SigHashNone | SigHashAnyOneCanPay, "88a34a6d44f1375d5d2ed42c28f559b5d69febb301430b3982d8a4d83502012f"},
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
