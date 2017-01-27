package bc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/errors"
	"chain/testutil"
)

func TestTransaction(t *testing.T) {
	issuanceScript := []byte{1}
	initialBlockHashHex := "03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"
	initialBlockHash := mustDecodeHash(initialBlockHashHex)

	assetID := ComputeAssetID(issuanceScript, initialBlockHash, 1, EmptyStringHash)

	cases := []struct {
		tx   *Tx
		hex  string
		hash [32]byte
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
			hash: mustDecodeHash("74e60d94a75848b48fc79eac11a1d39f41e1b32046cf948929b729a57b75d5be"),
		},
		{
			tx: NewTx(TxData{
				Version: 1,
				Inputs: []*TxInput{
					NewIssuanceInput([]byte{10, 9, 8}, 1000000000000, []byte("input"), initialBlockHash, issuanceScript, [][]byte{[]byte{1, 2, 3}}, nil),
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
				"2b" + // input 0, input commitment length prefix
				"00" + // input 0, input commitment, "issuance" type
				"03" + // input 0, input commitment, nonce length prefix
				"0a0908" + // input 0, input commitment, nonce
				assetID.String() + // input 0, input commitment, asset id
				"80a094a58d1d" + // input 0, input commitment, amount
				"05696e707574" + // input 0, reference data
				"29" + // input 0, issuance input witness length prefix
				initialBlockHashHex + // input 0, issuance input witness, initial block
				"00" + // input 0, issuance input witness, asset definition
				"01" + // input 0, issuance input witness, vm version
				"01" + // input 0, issuance input witness, issuance program length prefix
				"01" + // input 0, issuance input witness, issuance program
				"01" + // input 0, issuance input witness, arguments count
				"03" + // input 0, issuance input witness, argument 0 length prefix
				"010203" + // input 0, issuance input witness, argument 0
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
			hash: mustDecodeHash("946b6c226d88723020ab50665fbb76191639b5f1fd851b35edb729d4b17373cc"),
		},
		{
			tx: NewTx(TxData{
				Version: 1,
				Inputs: []*TxInput{
					NewSpendInput(ComputeOutputID(mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), 0), nil, AssetID{}, 1000000000000, []byte{1}, []byte("input")),
				},
				Outputs: []*TxOutput{
					NewTxOutput(ComputeAssetID(issuanceScript, initialBlockHash, 1, EmptyStringHash), 600000000000, []byte{1}, nil),
					NewTxOutput(ComputeAssetID(issuanceScript, initialBlockHash, 1, EmptyStringHash), 400000000000, []byte{2}, nil),
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
				"4b" + // input 0, input commitment length prefix
				"01" + // input 0, input commitment, "spend" type
				"110bd1b4e5efc2994c9abc77f223a52c834d8f26b907c6c19d90b9e77a8e2fed" + // input 0, spend input commitment, output ID
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
				"a9b2b6c5394888ab5396f583ae484b8459486b14268e2bef1b637440335eb6c1" + // output 0, output commitment, asset id
				"80e0a596bb11" + // output 0, output commitment, amount
				"01" + // output 0, output commitment, vm version
				"0101" + // output 0, output commitment, control program
				"00" + // output 0, reference data
				"00" + // output 0, output witness
				"01" + // output 1, asset version
				"29" + // output 1, output commitment length
				"a9b2b6c5394888ab5396f583ae484b8459486b14268e2bef1b637440335eb6c1" + // output 1, output commitment, asset id
				"80c0ee8ed20b" + // output 1, output commitment, amount
				"01" + // output 1, vm version
				"0102" + // output 1, output commitment, control program
				"00" + // output 1, reference data
				"00" + // output 1, output witness
				"0c646973747269627574696f6e"), // reference data
			hash: mustDecodeHash("86556ca6f6181dbb71bcd3cba53fce825f39083c409b7c4afdf40c9912487113"),
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

		txJSON, err := json.Marshal(test.tx)
		if err != nil {
			t.Errorf("test %d: error marshaling tx to json: %s", i, err)
		}
		var txFromJSON Tx
		if err := json.Unmarshal(txJSON, &txFromJSON); err != nil {
			t.Errorf("test %d: error unmarshaling tx from json: %s", i, err)
		}
		if !testutil.DeepEqual(test.tx, &txFromJSON) {
			t.Errorf("test %d: bc.Tx -> json -> bc.Tx: got:\n%s\nwant:\n%s", i, spew.Sdump(&txFromJSON), spew.Sdump(test.tx))
		}

		tx1 := new(TxData)
		if err := tx1.UnmarshalText([]byte(test.hex)); err != nil {
			t.Errorf("test %d: unexpected err %v", i, err)
		}
		if !testutil.DeepEqual(*tx1, test.tx.TxData) {
			t.Errorf("test %d: tx1 is:\n%swant:\n%s", i, spew.Sdump(*tx1), spew.Sdump(test.tx.TxData))
		}
	}
}

func TestHasIssuance(t *testing.T) {
	cases := []struct {
		tx   *TxData
		want bool
	}{{
		tx: &TxData{
			Inputs: []*TxInput{NewIssuanceInput(nil, 0, nil, Hash{}, nil, nil, nil)},
		},
		want: true,
	}, {
		tx: &TxData{
			Inputs: []*TxInput{
				NewSpendInput(ComputeOutputID(Hash{}, 0), nil, AssetID{}, 0, nil, nil),
				NewIssuanceInput(nil, 0, nil, Hash{}, nil, nil, nil),
			},
		},
		want: true,
	}, {
		tx: &TxData{
			Inputs: []*TxInput{
				NewSpendInput(ComputeOutputID(Hash{}, 0), nil, AssetID{}, 0, nil, nil),
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

func TestInvalidIssuance(t *testing.T) {
	hex := ("07" + // serflags
		"01" + // transaction version
		"02" + // common fields extensible string length
		"00" + // common fields, mintime
		"00" + // common fields, maxtime
		"00" + // common witness extensible string length
		"01" + // inputs count
		"01" + // input 0, asset version
		"2b" + // input 0, input commitment length prefix
		"00" + // input 0, input commitment, "issuance" type
		"03" + // input 0, input commitment, nonce length prefix
		"0a0908" + // input 0, input commitment, nonce
		"0000000000000000000000000000000000000000000000000000000000000000" + // input 0, input commitment, WRONG asset id
		"80a094a58d1d" + // input 0, input commitment, amount
		"05696e707574" + // input 0, reference data
		"29" + // input 0, issuance input witness length prefix
		"03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d" + // input 0, issuance input witness, initial block
		"00" + // input 0, issuance input witness, asset definition
		"01" + // input 0, issuance input witness, vm version
		"01" + // input 0, issuance input witness, issuance program length prefix
		"01" + // input 0, issuance input witness, issuance program
		"01" + // input 0, issuance input witness, arguments count
		"03" + // input 0, issuance input witness, argument 0 length prefix
		"010203" + // input 0, issuance input witness, argument 0
		"01" + // outputs count
		"01" + // output 0, asset version
		"29" + // output 0, output commitment length
		"0000000000000000000000000000000000000000000000000000000000000000" + // output 0, output commitment, asset id
		"80a094a58d1d" + // output 0, output commitment, amount
		"01" + // output 0, output commitment, vm version
		"0101" + // output 0, output commitment, control program
		"066f7574707574" + // output 0, reference data
		"00" + // output 0, output witness
		"0869737375616e6365")
	tx := new(TxData)
	err := tx.UnmarshalText([]byte(hex))
	if errors.Root(err) != errBadAssetID {
		t.Errorf("want errBadAssetID, got %v", err)
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
	assetID := ComputeAssetID([]byte{1}, mustDecodeHash("03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"), 1, EmptyStringHash)
	tx := &TxData{
		Version: 1,
		Inputs: []*TxInput{
			NewSpendInput(ComputeOutputID(mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151"), 0), [][]byte{[]byte{1}}, AssetID{}, 0, nil, []byte("input1")),
			NewSpendInput(ComputeOutputID(mustDecodeHash("d250fa36f2813ddb8aed0fc66790ee58121bcbe88909bf88be12083d45320151"), 1), [][]byte{[]byte{2}}, AssetID{}, 0, nil, nil),
		},
		Outputs: []*TxOutput{
			NewTxOutput(assetID, 1000000000000, []byte{3}, nil),
		},
		ReferenceData: []byte("transfer"),
	}
	cases := []struct {
		idx      uint32
		wantHash string
	}{
		// <------ 8fed9a267e73fcdf1c1f2fbcf6b90e2447dce5b8
		//      {0, "e5dcd964d94ce4aa8bf99e73df22d81f6b28fd54f4c2cb3c409a4cc7240cab49"},
		//      {1, "0359c90d5038adb93df55955782d350f43808a4dbd298f238861b7b505521c4e"},
		// -------
		{0, "aeee1f382653dfc09c89dcf18ad8b641ff6ab05450352798a16f0ef4a4026618"},
		{1, "c25e5b4bbf6e440525cda4a77ca374c9cca14d0918d4b4cc8635db9c5ee1f5f3"},
	}

	sigHasher := NewSigHasher(tx)

	for _, c := range cases {
		hash := tx.HashForSig(c.idx)

		if hash.String() != c.wantHash {
			t.Errorf("HashForSig(%d) = %s want %s", c.idx, hash.String(), c.wantHash)
		}

		cachedHash := sigHasher.Hash(c.idx)

		if cachedHash.String() != c.wantHash {
			t.Errorf("sigHasher.Hash(%d) = %s want %s", c.idx, hash.String(), c.wantHash)
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
		tx.Inputs = append(tx.Inputs, NewSpendInput(ComputeOutputID(Hash{}, 0), nil, AssetID{}, 0, nil, nil))
		tx.Outputs = append(tx.Outputs, NewTxOutput(AssetID{}, 0, nil, nil))
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, 0)
	}
}

func BenchmarkTxWriteToFalse200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, NewSpendInput(ComputeOutputID(Hash{}, 0), nil, AssetID{}, 0, nil, nil))
		tx.Outputs = append(tx.Outputs, NewTxOutput(AssetID{}, 0, nil, nil))
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, serRequired)
	}
}

func BenchmarkTxInputWriteToTrue(b *testing.B) {
	input := NewSpendInput(ComputeOutputID(Hash{}, 0), nil, AssetID{}, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew, 0)
	}
}

func BenchmarkTxInputWriteToFalse(b *testing.B) {
	input := NewSpendInput(ComputeOutputID(Hash{}, 0), nil, AssetID{}, 0, nil, nil)
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

func BenchmarkAssetAmountWriteTo(b *testing.B) {
	aa := AssetAmount{}
	for i := 0; i < b.N; i++ {
		aa.writeTo(ioutil.Discard)
	}
}

func BenchmarkOutpointWriteTo(b *testing.B) {
	o := Outpoint{}
	for i := 0; i < b.N; i++ {
		o.WriteTo(ioutil.Discard)
	}
}
