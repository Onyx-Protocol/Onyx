package bc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	"chain/errors"
)

var now = time.Unix(233400000, 0)

func TestTransaction(t *testing.T) {
	issuanceScript := []byte{1}
	genesisHashHex := "03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"
	genesisHash := mustDecodeHash(genesisHashHex)

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
			witnessHash: mustDecodeHash("372a82e424cdadc1d233e6a000b5b644b84472a6eb8365d23f052cb8663e2ff7"),
		},
		{
			tx: NewTx(TxData{
				SerFlags: 0x7,
				Version:  1,
				Inputs: []*TxInput{
					NewIssuanceInput(now, now.Add(time.Hour), genesisHash, 1000000000000, issuanceScript, nil, []byte("input"), [][]byte{[]byte{1, 2, 3}}),
				},
				Outputs: []*TxOutput{
					NewTxOutput(AssetID{}, 1000000000000, []byte{1}, []byte("output")),
				},
				MinTime:  0,
				MaxTime:  0,
				Metadata: []byte("issuance"),
			}),
			hex: ("07" + // serflags
				"01" + // transaction version
				"01" + // inputs count
				"01" + // input 0, asset version
				"37" + // input 0, input commitment length prefix
				"00" + // input 0, input commitment, "issuance" type
				"80bce5bde506" + // input 0, input commitment, mintime
				"8099c1bfe506" + // input 0, input commitment, maxtime
				genesisHashHex + // input 0, input commitment, initial block
				"80a094a58d1d" + // input 0, input commitment, amount
				"01" + // input 0, input commitment, vm version
				"0101" + // input 0, input commitment, issuance program
				"00" + // input 0, input commitment, asset definition
				"05696e707574" + // input 0, reference data
				"05" + // input 0, input witness length prefix
				"01" + // input 0, input witness, number of args
				"03010203" + // input 0, input witness, arg 0
				"01" + // outputs count
				"01" + // output 0, asset version
				"29" + // output 0, output commitment length
				"0000000000000000000000000000000000000000000000000000000000000000" + // output 0, output commitment, asset id
				"80a094a58d1d" + // output 0, output commitment, amount
				"01" + // output 0, output commitment, vm version
				"0101" + // output 0, output commitment, control program
				"066f7574707574" + // output 0, reference data
				"00" + // output 0, output witness
				"00" + // mintime
				"00" + // maxtime
				"0869737375616e6365"), // reference data
			hash:        mustDecodeHash("4b3c0036f7de199c41e3d9b993b14d4482d9351beed538e0fc4e1cc56a60e1b6"),
			witnessHash: mustDecodeHash("cfa2be43281d536bf73027fcf157609d9006fd808bf3e1d89de94489e554789c"),
		},
		{
			tx: NewTx(TxData{
				SerFlags: 0x7,
				Version:  1,
				Inputs: []*TxInput{
					NewSpendInput(mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), 0, nil, AssetID{}, 1000000000000, []byte{1}, []byte("input")),
				},
				Outputs: []*TxOutput{
					NewTxOutput(ComputeAssetID(issuanceScript, genesisHash, 1), 600000000000, []byte{1}, nil),
					NewTxOutput(ComputeAssetID(issuanceScript, genesisHash, 1), 400000000000, []byte{2}, nil),
				},
				MinTime:  1492590000,
				MaxTime:  1492590591,
				Metadata: []byte("distribution"),
			}),
			hex: ("07" + // serflags
				"01" + // transaction version
				"01" + // inputs count
				"01" + // input 0, asset version
				"4c" + // input 0, input commitment length prefix
				"01" + // input 0, input commitment, "spend" type
				"dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292" + // input 0, spend input commitment, outpoint tx hash
				"00" + // input 0, spend input commitment, outpoint index
				"29" + // input 0, spend input commitment, output commitment length prefix
				"0000000000000000000000000000000000000000000000000000000000000000" + // input 0, spend input commitment, output commitment, asset id
				"80a094a58d1d" + // input 0, spend input commitment, output commitment, amount
				"01" + // input 0, spend input commitment, output commitment, vm version
				"0101" + // input 0, spend input commitment, output commitment, control program
				"05696e707574" + // input 0, reference data
				"01" + // input 0, input witness length prefix
				"00" + // input 0, input witness, number of args
				"02" + // outputs count
				"01" + // output 0, asset version
				"29" + // output 0, output commitment length
				"9ed3e85a8c2d3717b5c94bd2db2ab9cab56955b2c4fb4696f345ca97aaab82d6" + // output 0, output commitment, asset id
				"80e0a596bb11" + // output 0, output commitment, amount
				"01" + // output 0, output commitment, vm version
				"0101" + // output 0, output commitment, control program
				"00" + // output 0, reference data
				"00" + // output 0, output witness
				"01" + // output 1, asset version
				"29" + // output 1, output commitment length
				"9ed3e85a8c2d3717b5c94bd2db2ab9cab56955b2c4fb4696f345ca97aaab82d6" + // output 1, output commitment, asset id
				"80c0ee8ed20b" + // output 1, output commitment, amount
				"01" + // output 1, vm version
				"0102" + // output 1, output commitment, control program
				"00" + // output 1, reference data
				"00" + // output 1, output witness
				"b0bbdcc705" + // mintime
				"ffbfdcc705" + // maxtime
				"0c646973747269627574696f6e"), // reference data
			hash:        mustDecodeHash("fcbd7e149d5db32bc7635cd313e9de37fcd24e01492f057ece36a799555c5dee"),
			witnessHash: mustDecodeHash("615689585f1f882d76e1d50f2fd719bb433c25992d4833f4de696824b792a8f2"),
		},
	}

	for i, test := range cases {
		got := serialize(t, test.tx)
		want, _ := hex.DecodeString(test.hex)
		if !bytes.Equal(got, want) {
			t.Errorf("test %d: bytes = %x want %x", i, got, want)
		}
		if test.tx.Hash != test.hash {
			t.Errorf("test %d: hash = %s want %x", i, test.tx.Hash, test.hash)
		}
		if g := test.tx.WitnessHash(); g != test.witnessHash {
			t.Errorf("test %d: witness hash = %s want %x", i, g, test.witnessHash)
		}

		txJSON, err := json.Marshal(test.tx)
		if err != nil {
			t.Errorf("test %d: error marshaling tx to json: %s", i, err)
		}
		var txFromJSON Tx
		if err := json.Unmarshal(txJSON, &txFromJSON); err != nil {
			t.Errorf("test %d: error unmarshaling tx from json: %s", i, err)
		}
		if !reflect.DeepEqual(test.tx, &txFromJSON) {
			t.Errorf("test %d: bc.Tx -> json -> bc.Tx: got:\n%s\nwant:\n%s", i, spew.Sdump(&txFromJSON), spew.Sdump(test.tx))
		}

		tx1 := new(TxData)
		if err := tx1.UnmarshalText([]byte(test.hex)); err != nil {
			t.Errorf("test %d: unexpected err %v", i, err)
		}
		if !reflect.DeepEqual(*tx1, test.tx.TxData) {
			t.Errorf("test %d: tx1 = %v want %v", i, *tx1, test.tx.TxData)
		}
	}
}

func TestHasIssuance(t *testing.T) {
	cases := []struct {
		tx   *TxData
		want bool
	}{{
		tx: &TxData{
			Inputs: []*TxInput{NewIssuanceInput(now, now.Add(time.Hour), Hash{}, 0, nil, nil, nil, nil)},
		},
		want: true,
	}, {
		tx: &TxData{
			Inputs: []*TxInput{
				NewSpendInput(Hash{}, 0, nil, AssetID{}, 0, nil, nil),
				NewIssuanceInput(now, now.Add(time.Hour), Hash{}, 0, nil, nil, nil, nil),
			},
		},
		want: true,
	}, {
		tx: &TxData{
			Inputs: []*TxInput{
				NewSpendInput(Hash{}, 0, nil, AssetID{}, 0, nil, nil),
			},
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
	assetID := ComputeAssetID([]byte{1}, mustDecodeHash("03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"), 1)
	tx := &TxData{
		SerFlags: 0x7,
		Version:  1,
		Inputs: []*TxInput{
			NewSpendInput(mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151"), 0, [][]byte{[]byte{1}}, AssetID{}, 0, nil, []byte("input1")),
			NewSpendInput(mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151"), 1, [][]byte{[]byte{2}}, AssetID{}, 0, nil, nil),
		},
		Outputs: []*TxOutput{
			NewTxOutput(assetID, 1000000000000, []byte{3}, nil),
		},
		Metadata: []byte("transfer"),
	}
	cases := []struct {
		idx      int
		hashType SigHashType
		wantHash string
	}{
		// TODO(bobg): Update all these hashes to pass under new serialization logic in PR 1070 (and possibly others)
		{0, SigHashAll, "f63aea25b373b683c367f3dbc94c25ac86b374e2e6f9d01c7e0f4f7465ff48f4"},
		{0, SigHashSingle, "cd9cb71dd1adf22afc0b4c5eed82b35cd33fdb2cc0e86bbe52e86e0801f90caa"},
		{0, SigHashNone, "46d93487fcd01b528d0eecab1b22c7e5eef29e63cccc7285be666b6f3c90ab20"},
		{0, SigHashAll | SigHashAnyOneCanPay, "8d273a2c1a78129aa87dd914e5ea7b90f15946d7b7cf91e95177f1c53593a685"},
		{0, SigHashSingle | SigHashAnyOneCanPay, "cdb627c065ebb30de501d8b159038b5ae98fc404219ad53731b7ac1a91bd0ff4"},
		{0, SigHashNone | SigHashAnyOneCanPay, "e79409194a99fc3d5af4eea63ea1d4fca8fa14306eb1702ee09b05fc3d04ee3f"},

		{1, SigHashAll, "95f89e47c7ae3eed075763a2363e846e3450ae32267286126462adf03a054748"},
		{1, SigHashSingle, "332a73ae16be1233c22d5756f5982640a03fd6eb3f8473801d92ff3a3c78015c"},
		{1, SigHashNone, "8355a71b8a2aa9b8b5f8959bb3201365536112ded1f8c4706722918e68ab1b41"},
		{1, SigHashAll | SigHashAnyOneCanPay, "c9550b707b5d71d5b9fa9fc8a738cc79282e489478e292094cf07a56eb4975b9"},
		{1, SigHashSingle | SigHashAnyOneCanPay, "13e3f0ccaa667b723494dfd253e0a7f43c40f781586e8325f8fbafcf8f6a934e"},
		{1, SigHashNone | SigHashAnyOneCanPay, "13c2071104b0409cd6d73acd8d291d7c79745d5799969e113dadb7cc8c2fecb7"},
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
		tx.Inputs = append(tx.Inputs, NewSpendInput(Hash{}, 0, nil, AssetID{}, 0, nil, nil))
		tx.Outputs = append(tx.Outputs, NewTxOutput(AssetID{}, 0, nil, nil))
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, 0)
	}
}

func BenchmarkTxWriteToFalse200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, NewSpendInput(Hash{}, 0, nil, AssetID{}, 0, nil, nil))
		tx.Outputs = append(tx.Outputs, NewTxOutput(AssetID{}, 0, nil, nil))
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, serRequired)
	}
}

func BenchmarkTxInputWriteToTrue(b *testing.B) {
	input := NewSpendInput(Hash{}, 0, nil, AssetID{}, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew, 0)
	}
}

func BenchmarkTxInputWriteToFalse(b *testing.B) {
	input := NewSpendInput(Hash{}, 0, nil, AssetID{}, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew, serRequired)
	}
}

func BenchmarkTxOutputWriteToTrue(b *testing.B) {
	output := NewTxOutput(AssetID{}, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew, 0)
	}
}

func BenchmarkTxOutputWriteToFalse(b *testing.B) {
	output := NewTxOutput(AssetID{}, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew, serRequired)
	}
}
