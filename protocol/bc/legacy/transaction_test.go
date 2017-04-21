package legacy

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/errors"
	"chain/protocol/bc"
	"chain/testutil"
)

func TestTransactionTrailingGarbage(t *testing.T) {
	const validTxHex = `07010700d0929893b92b00000101270eac870dfde1e0feaa4fac6693dee38da2afe7f5cc83ce2b024f04a2400fd6e20a0104deadbeef027b7d0000`

	var validTx Tx
	err := validTx.UnmarshalText([]byte(validTxHex))
	if err != nil {
		t.Fatal(err)
	}

	invalidTxHex := validTxHex + strings.Repeat("beef", 10)
	var invalidTx Tx
	err = invalidTx.UnmarshalText([]byte(invalidTxHex))
	if err == nil {
		t.Fatal("expected error with trailing garbage but got nil")
	}
}

func TestTransaction(t *testing.T) {
	issuanceScript := []byte{1}
	initialBlockHashHex := "03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d"
	initialBlockHash := mustDecodeHash(initialBlockHashHex)

	assetID := bc.ComputeAssetID(issuanceScript, &initialBlockHash, 1, &bc.EmptyStringHash)

	cases := []struct {
		tx   *Tx
		hex  string
		hash bc.Hash
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
			hash: mustDecodeHash("7ae6eef6b02fe61d35cc185405aec5f690ccb0ac291ecd6214445a1dff8fc9fd"),
		},
		{
			tx: NewTx(TxData{
				Version: 1,
				Inputs: []*TxInput{
					NewIssuanceInput([]byte{10, 9, 8}, 1000000000000, []byte("input"), initialBlockHash, issuanceScript, [][]byte{[]byte{1, 2, 3}}, nil),
				},
				Outputs: []*TxOutput{
					NewTxOutput(bc.AssetID{}, 1000000000000, []byte{1}, []byte("output")),
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
			hash: mustDecodeHash("cd4669d5363374f8661621273501c23e613fc98b0fab9d5d858f30e16ccd24ce"),
		},
		{
			tx: NewTx(TxData{
				Version: 1,
				Inputs: []*TxInput{
					NewSpendInput(nil, mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), bc.AssetID{}, 1000000000000, 1, []byte{1}, bc.Hash{}, []byte("input")),
				},
				Outputs: []*TxOutput{
					NewTxOutput(assetID, 600000000000, []byte{1}, nil),
					NewTxOutput(assetID, 400000000000, []byte{2}, nil),
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
				"6c" + // input 0, input commitment length prefix
				"01" + // input 0, input commitment, "spend" type+
				"6a" + // input 0, spend input commitment, spend commitment length prefix
				"dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292" + // input 0, spend input commitment, spend commitment, source ID
				"0000000000000000000000000000000000000000000000000000000000000000" + // input 0, spend input commitment, spend commitment, asset id
				"80a094a58d1d" + // input 0, spend input commitment, spend commitment, amount
				"01" + // input 0, spend input commitment, spend commitment, source position
				"01" + // input 0, spend input commitment, spend commitment, vm version
				"0101" + // input 0, spend input commitment, spend commitment, control program
				"0000000000000000000000000000000000000000000000000000000000000000" + // input 0, spend input commitment, spend commitment, reference data hash
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
			hash: mustDecodeHash("c328ad4278045b4c50e8af7e7d0df198e7d9436d2b5de35df1339f13a1192331"),
		},

		//07
		//01
		//0a
		//b0bbdcc705
		//ffbfdcc705
		//00
		//01
		//01
		//4b
		//01
		//dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292
		//29
		//0000000000000000000000000000000000000000000000000000000000000000
		//80a094a58d1d
		//01
		//0101
		//05696e707574
		//01
		//00
		//02
		//01
		//29
		//a9b2b6c5394888ab5396f583ae484b8459486b14268e2bef1b637440335eb6c1
		//80e0a596bb11
		//01
		//0101
		//00
		//00
		//01
		//29
		//a9b2b6c5394888ab5396f583ae484b8459486b14268e2bef1b637440335eb6c1
		//80c0ee8ed20b
		//01
		//0102
		//00
		//00
		//0c646973747269627574696f6e

	}
	for i, test := range cases {
		got := serialize(t, test.tx)
		want, _ := hex.DecodeString(test.hex)
		if !bytes.Equal(got, want) {
			t.Errorf("test %d: bytes = %x want %x", i, got, want)
		}
		if test.tx.ID != test.hash {
			t.Errorf("test %d: hash = %x want %x", i, test.tx.ID.Bytes(), test.hash.Bytes())
		}

		txJSON, err := json.Marshal(test.tx)
		if err != nil {
			t.Errorf("test %d: error marshaling tx to json: %s", i, err)
		}
		var txFromJSON Tx
		if err := json.Unmarshal(txJSON, &txFromJSON); err != nil {
			t.Errorf("test %d: error unmarshaling tx from json: %s", i, err)
		}
		if !testutil.DeepEqual(test.tx.TxData, txFromJSON.TxData) {
			t.Errorf("test %d: legacy.TxData -> json -> legacy.TxData: got:\n%s\nwant:\n%s", i, spew.Sdump(txFromJSON.TxData), spew.Sdump(test.tx.TxData))
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
			Inputs: []*TxInput{NewIssuanceInput(nil, 0, nil, bc.Hash{}, nil, nil, nil)},
		},
		want: true,
	}, {
		tx: &TxData{
			Inputs: []*TxInput{
				NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil, bc.Hash{}, nil),
				NewIssuanceInput(nil, 0, nil, bc.Hash{}, nil, nil, nil),
			},
		},
		want: true,
	}, {
		tx: &TxData{
			Inputs: []*TxInput{
				NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil, bc.Hash{}, nil),
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
		tx.Inputs = append(tx.Inputs, NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil, bc.Hash{}, nil))
		tx.Outputs = append(tx.Outputs, NewTxOutput(bc.AssetID{}, 0, nil, nil))
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, 0)
	}
}

func BenchmarkTxWriteToFalse200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil, bc.Hash{}, nil))
		tx.Outputs = append(tx.Outputs, NewTxOutput(bc.AssetID{}, 0, nil, nil))
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, serRequired)
	}
}

func BenchmarkTxInputWriteToTrue(b *testing.B) {
	input := NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil, bc.Hash{}, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew, 0)
	}
}

func BenchmarkTxInputWriteToFalse(b *testing.B) {
	input := NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil, bc.Hash{}, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew, serRequired)
	}
}

func BenchmarkTxOutputWriteToTrue(b *testing.B) {
	output := NewTxOutput(bc.AssetID{}, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew, 0)
	}
}

func BenchmarkTxOutputWriteToFalse(b *testing.B) {
	output := NewTxOutput(bc.AssetID{}, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew, serRequired)
	}
}

func BenchmarkAssetAmountWriteTo(b *testing.B) {
	aa := bc.AssetAmount{}
	for i := 0; i < b.N; i++ {
		aa.WriteTo(ioutil.Discard)
	}
}
