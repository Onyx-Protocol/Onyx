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
				Version:       1,
				Inputs:        nil,
				Outputs:       nil,
				MinTime:       0,
				MaxTime:       0,
				ReferenceData: nil,
			}),
			hex: ("07" + // serflags
				"01" + // transaction version
				"02" + // common fields extensible string length
				"00" + // common fields, mintime
				"00" + // common fields, maxtime
				"00" + // common witness extensible string length
				"00" + // inputs count
				"00" + // outputs count
				"00"), // reference data
			hash:        mustDecodeHash("74e60d94a75848b48fc79eac11a1d39f41e1b32046cf948929b729a57b75d5be"),
			witnessHash: mustDecodeHash("536cef3158d7ea51194b370e02f27265e8584ff4df1cd2829de0074c11f1f1b2"),
		},
		{
			tx: NewTx(TxData{
				Version: 1,
				Inputs: []*TxInput{
					NewIssuanceInput(now, now.Add(time.Hour), genesisHash, 1000000000000, issuanceScript, []byte("input"), [][]byte{[]byte{1, 2, 3}}),
				},
				Outputs: []*TxOutput{
					NewTxOutput(AssetID{}, 1000000000000, []byte{1}, []byte("output")),
				},
				MinTime:       0,
				MaxTime:       0,
				ReferenceData: []byte("issuance"),
			}),
			hex: ("07" + // serflags
				"01" + // transaction version
				"02" + // common fields extensible string length
				"00" + // common fields, mintime
				"00" + // common fields, maxtime
				"00" + // common witness extensible string length
				"01" + // inputs count
				"01" + // input 0, asset version
				"36" + // input 0, input commitment length prefix
				"00" + // input 0, input commitment, "issuance" type
				"80bce5bde506" + // input 0, input commitment, mintime
				"8099c1bfe506" + // input 0, input commitment, maxtime
				genesisHashHex + // input 0, input commitment, initial block
				"80a094a58d1d" + // input 0, input commitment, amount
				"01" + // input 0, input commitment, vm version
				"0101" + // input 0, input commitment, issuance program
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
				"0869737375616e6365"), // reference data
			hash:        mustDecodeHash("efa7ded16f69f183f84dd0dcb9108d6b41c66ce91d2d05853e32a5a42306aee5"),
			witnessHash: mustDecodeHash("a0560a79ebc2fa623a61afe8914242f89daf60884ab0dcede37f4251e0f5de0f"),
		},
		{
			tx: NewTx(TxData{
				Version: 1,
				Inputs: []*TxInput{
					NewSpendInput(mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), 0, nil, AssetID{}, 1000000000000, []byte{1}, []byte("input")),
				},
				Outputs: []*TxOutput{
					NewTxOutput(ComputeAssetID(issuanceScript, genesisHash, 1), 600000000000, []byte{1}, nil),
					NewTxOutput(ComputeAssetID(issuanceScript, genesisHash, 1), 400000000000, []byte{2}, nil),
				},
				MinTime:       1492590000,
				MaxTime:       1492590591,
				ReferenceData: []byte("distribution"),
			}),
			hex: ("07" + // serflags
				"01" + // transaction version
				"0a" + // common fields extensible string length
				"b0bbdcc705" + // common fields, mintime
				"ffbfdcc705" + // common fields, maxtime
				"00" + // common witness extensible string length
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
				"0c646973747269627574696f6e"), // reference data
			hash:        mustDecodeHash("b752c2c6e423fb228d5d4f3b4bc2cf008317608f03c156d9b8c950b058659a38"),
			witnessHash: mustDecodeHash("3bcdf9c8c8285da9252604708340b4cc9c2f4ae4268481ff1910d972cfd7438e"),
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
			Inputs: []*TxInput{NewIssuanceInput(now, now.Add(time.Hour), Hash{}, 0, nil, nil, nil)},
		},
		want: true,
	}, {
		tx: &TxData{
			Inputs: []*TxInput{
				NewSpendInput(Hash{}, 0, nil, AssetID{}, 0, nil, nil),
				NewIssuanceInput(now, now.Add(time.Hour), Hash{}, 0, nil, nil, nil),
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
		Version: 1,
		Inputs: []*TxInput{
			NewSpendInput(mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151"), 0, [][]byte{[]byte{1}}, AssetID{}, 0, nil, []byte("input1")),
			NewSpendInput(mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151"), 1, [][]byte{[]byte{2}}, AssetID{}, 0, nil, nil),
		},
		Outputs: []*TxOutput{
			NewTxOutput(assetID, 1000000000000, []byte{3}, nil),
		},
		ReferenceData: []byte("transfer"),
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
