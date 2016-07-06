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
				MinTime:  0,
				MaxTime:  0,
				Metadata: nil,
			}),
			hex: ("07" + // serflags
				"01" + // transaction version
				"00" + // inputs count
				"00" + // outputs count
				"00" + // mintime
				"00" + // maxtime
				"00"), // reference data
			hash:        mustDecodeHash("8a25dbad170e0e36fe6ef5c4479b44c7a5ec03d300a693671bb6c851a7ade2e3"),
			witnessHash: mustDecodeHash("ac2e154a278bf41763d3dc5e2a5fbafc03d427e882fbe04297d6dc06509f3bcc"),
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
				MinTime:  0,
				MaxTime:  0,
				Metadata: []byte("issuance"),
			}),
			hex: ("07" + // serflags
				"01" + // transaction version
				"01" + // inputs count
				"03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d" + // input 0, spend input commitment, outpoint tx hash
				"ffffffff0f" + // input 0, spend input commitment, outpoint index
				"0000000000000000000000000000000000000000000000000000000000000000" + // input 0, output commitment, asset id
				"00" + // input 0, output commitment, amount
				"00" + // input 0, output commitment, control program
				"03010203" + // input 0, input witness, sigscript
				"05696e707574" + // input 0, reference data
				"00" + // input 0, asset definition
				"01" + // outputs count
				"0000000000000000000000000000000000000000000000000000000000000000" + // output 0, output commitment, asset id
				"80a094a58d1d" + // output 0, output commitment, amount
				"0101" + // output 0, output commitment, control program
				"066f7574707574" + // output 0, reference data
				"00" + // mintime
				"00" + // maxtime
				"0869737375616e6365"), // reference data
			hash:        mustDecodeHash("d13bbc2c411c470c335a8be5f11e4d97badbd98e54a27250701af73723bd4671"),
			witnessHash: mustDecodeHash("a0ad9dbe92de70ac361e097053fda7080bd31652c1d545573032bb7de8779d27"),
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
				MinTime:  1492590000,
				MaxTime:  1492590591,
				Metadata: []byte("distribution"),
			}),
			hex: ("07" + // serflags
				"01" + // transaction version
				"01" + // inputs count
				"dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292" + // input 0, spend input commitment, outpoint tx hash
				"00" + // input 0, spend input commitment, outpoint index
				"0000000000000000000000000000000000000000000000000000000000000000" + // input 0, output commitment, asset id
				"80a094a58d1d" + // input 0, output commitment, amount
				"0101" + // input 0, output commitment, control program
				"00" + // input 0, input witness, sigscript
				"05696e707574" + // input 0, reference data
				"086173736574646566" + // input 0, asset definition
				"02" + // outputs count
				"8ce7bfa83eeb157470101b2c40d528335bf9e98c9383f6f6e575bee3e2131236" + // output 0, output commitment, asset id
				"80e0a596bb11" + // output 0, output commitment, amount
				"0101" + // output 0, output commitment, control program
				"00" + // output 0, reference data
				"8ce7bfa83eeb157470101b2c40d528335bf9e98c9383f6f6e575bee3e2131236" + // output 1, output commitment, asset id
				"80c0ee8ed20b" + // output 1, output commitment, amount
				"0102" + // output 1, output commitment, control program
				"00" + // output 1, reference data
				"b0bbdcc705" + // mintime
				"ffbfdcc705" + // maxtime
				"0c646973747269627574696f6e"), // reference data
			hash:        mustDecodeHash("ce3ec06d9bd26c5ff2c6ec47314312f8cfea809a37c12beb14a6e98315e02de0"),
			witnessHash: mustDecodeHash("1996737c639822f3d15d70e425df2f77697f29bc61f686264a1059a9e4634e54"),
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
		{0, SigHashAll, "ad34a69f12985aef8d54cd5159d3048f7737bed6c2f023a38243fc69503b78b5"},
		{0, SigHashSingle, "a5eb4219eb97a38c8cb10c9cb5c62d5b7f168b15a197074195b1b10e40da57de"},
		{0, SigHashNone, "afcfc807b05ba0359425bed9cbc134e816cdd9fde6ecc264d0ac1a2c77687377"},
		{0, SigHashAll | SigHashAnyOneCanPay, "c2e337650b6ab02f900a6431185741824f7398cbd03e44dc481368549bef1820"},
		{0, SigHashSingle | SigHashAnyOneCanPay, "ee4170aaf0557225e087c563f437770445be7cbdcb6465b25d5c2c9f27508aca"},
		{0, SigHashNone | SigHashAnyOneCanPay, "76d02d4c31c4feea2a90a541ec3614871ecd2be8e0bb977c5b7f803fa13ad9bc"},

		{1, SigHashAll, "4b65fbebc2a929dff3a0e39a9ea93334cfad82f21d52681b1f549827025c82c4"},
		{1, SigHashSingle, "5660dc159e5c893085b214b96d9f557b4ef66ecf26b22efdead512578a001998"},
		{1, SigHashNone, "6a705d7618f2b1f56b4f07c765ffb1fa63e9ae1fe81ecd106454e6c267e1ba84"},
		{1, SigHashAll | SigHashAnyOneCanPay, "c20e235eb69149b3ea980be67fee440ecd260b4331551519463eab1bbd44106f"},
		{1, SigHashSingle | SigHashAnyOneCanPay, "81deccfe443e8c150307727337c16427a6140583a9172e3c0be65fb57a695a14"},
		{1, SigHashNone | SigHashAnyOneCanPay, "e8fd4239f7ee14c9e464ca8b2c154e1367993b0bb38b569adbda14886aeff149"},
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
