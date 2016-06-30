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
				SerFlags: 0x7,
				Version:  1,
				Inputs:   nil,
				Outputs:  nil,
				LockTime: 0,
				Metadata: nil,
			}),
			hex:         "07010000000000000000000000000000",
			hash:        mustDecodeHash("1165aef97cc72a200ef401440c7ce03514c0e2df74bf25e450cfe2da0ef99897"),
			witnessHash: mustDecodeHash("42a6d66913142d80deabee63cddd87408b6959f4b80dc559ab892fe0ba21631c"),
		},
		{
			tx: NewTx(TxData{
				SerFlags: 0x7,
				Version:  1,
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
			hex:         "07010000000103deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758dffffffff00000000000000000000000000000000000000000000000000000000000000000000000000000000000301020305696e707574000100000000000000000000000000000000000000000000000000000000000000000010a5d4e80000000101066f757470757400000000000000000869737375616e6365",
			hash:        mustDecodeHash("39d3ac49f3a3c3dbcd8c780ddac4a37d84a40c194b80ae0267df61120844fbba"),
			witnessHash: mustDecodeHash("0e495e4ef07e208123aa90090f224360809b4447236be4c0217e31c907cf9ecf"),
		},
		{
			tx: NewTx(TxData{
				SerFlags: 0x7,
				Version:  1,
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
			hex:         "070100000001dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d32920000000000000000000000000000000000000000000000000000000000000000000000000010a5d4e800000001010005696e707574086173736574646566028ce7bfa83eeb157470101b2c40d528335bf9e98c9383f6f6e575bee3e21312360070c9b28b0000000101008ce7bfa83eeb157470101b2c40d528335bf9e98c9383f6f6e575bee3e213123600a0db215d000000010200ff1ff758000000000c646973747269627574696f6e",
			hash:        mustDecodeHash("138c6a817b6bbd1fd4cbbb570f72b434e9e2e06a77655377e76b273a0047c5b0"),
			witnessHash: mustDecodeHash("b501ae20c7894b623d5e3b802b48f307d80d7ad06a5d3eb8bbc064fed40c0009"),
		},
	}

	for _, test := range cases {
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
		SerFlags: 0x7,
		Version:  1,
		Inputs: []*TxInput{
			{
				Previous:        Outpoint{Hash: mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151")},
				SignatureScript: []byte{1},
				Metadata:        []byte("input1"),
			},
			{
				Previous:        Outpoint{Hash: mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151"), Index: 1},
				SignatureScript: []byte{2},
			},
		},
		Outputs: []*TxOutput{
			{
				AssetAmount: AssetAmount{AssetID: assetID, Amount: 1000000000000},
				Script:      []byte{3},
			},
		},
		Metadata: []byte("transfer"),
	}
	cases := []struct {
		idx      int
		hashType SigHashType
		wantHash string
	}{
		// TODO(bobg): Update all these hashes to pass under new serialization logic in PR 1070 (and possibly others)
		{0, SigHashAll, "85b62dc7b0352882940a2589b0117a391efb424080f0d58436d00fda785cade7"},
		{0, SigHashSingle, "a39a403c0662acce9dc61c2ba0502cfd038ae127408830567775cfd7d506fbf2"},
		{0, SigHashNone, "dd7965880364b729f751bc9f285d63048fb558387e6bcea5f0a63f1bd1f8b2f8"},
		{0, SigHashAll | SigHashAnyOneCanPay, "309f36191b469d9d022694cf3d3c951741c298d753e1b41faa1877fa09bf4f1d"},
		{0, SigHashSingle | SigHashAnyOneCanPay, "329d5bbcb374be8783e4a77f90314d0d83d119b51b8e99a6e3df0e0a8e36ba20"},
		{0, SigHashNone | SigHashAnyOneCanPay, "bf3e1991bf2c176d06fb4c049aa7825146fb5e6d2eda4af0dac584a2376af5f1"},

		{1, SigHashAll, "c637509cc4e166848f25ab5cfef5cb35e46961042bd020f05be94f98ef2f75f5"},
		{1, SigHashSingle, "d778f3b77c562d621e9ce443d3a93d86776538316ddb2bef8885efb729ee2c66"},
		{1, SigHashNone, "8823a53ba361d73e86adb042104ca46c38bd3c5597fee4ea5f2b3d2c47edd152"},
		{1, SigHashAll | SigHashAnyOneCanPay, "3a9493f33b36a907ff38db9fafbacb06afcc99d2b5152de850e94172d004c99c"},
		{1, SigHashSingle | SigHashAnyOneCanPay, "b9529009522ab8923755b598b59f0b7e42a2042a0a6f4749c409192fed1c5141"},
		{1, SigHashNone | SigHashAnyOneCanPay, "c97f85e80ed62dd775f41b145ec4a94829352df1eef8cbf7e7f785fc7c508929"},
	}

	sigHasher := NewSigHasher(tx)

	for _, c := range cases {
		hash := tx.HashForSig(c.idx, c.hashType)

		if hash.String() != c.wantHash {
			t.Errorf("HashForSig(%d, %v) = %s want %s", c.idx, c.hashType, hash.String(), c.wantHash)
		}

		cachedHash := sigHasher.Hash(c.idx, c.hashType)

		if cachedHash.String() != c.wantHash {
			t.Errorf("sigHasher.Hash(%d, %v) = %s want %s", c.idx, c.hashType, hash.String(), c.wantHash)
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
		tx.writeTo(ioutil.Discard, 0)
	}
}

func BenchmarkTxWriteToFalse(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, serRequired)
	}
}

func BenchmarkTxWriteToTrue200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, &TxInput{})
		tx.Outputs = append(tx.Outputs, &TxOutput{})
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, 0)
	}
}

func BenchmarkTxWriteToFalse200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, &TxInput{})
		tx.Outputs = append(tx.Outputs, &TxOutput{})
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, serRequired)
	}
}

func BenchmarkTxInputWriteToTrue(b *testing.B) {
	input := &TxInput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew, 0)
	}
}

func BenchmarkTxInputWriteToFalse(b *testing.B) {
	input := &TxInput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew, serRequired)
	}
}

func BenchmarkTxOutputWriteToTrue(b *testing.B) {
	output := &TxOutput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew, 0)
	}
}

func BenchmarkTxOutputWriteToFalse(b *testing.B) {
	output := &TxOutput{}
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew, serRequired)
	}
}
