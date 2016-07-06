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
			hex: ("07" + // serflags
				"01" + // transaction version
				"00" + // inputs count
				"00" + // outputs count
				"00" + // locktime
				"00"), // reference data
			hash:        mustDecodeHash("21fe00fff828f20b73ab5502c21870952c5d8b01f90fddf81e4ad2d629590b9c"),
			witnessHash: mustDecodeHash("448ce15b340978d51701fed76d5adbb69bab6ae0abea617a7ce565734266bdb9"),
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
				"00" + // locktime
				"0869737375616e6365"), // reference data
			hash:        mustDecodeHash("ea83ace74146267c36f6cf818b544c5e96019484b2b02c8044a46176eef09c03"),
			witnessHash: mustDecodeHash("035383153eccc5d0216cde264ff252763c348f3a17d0c3b14cb96a4ddb4e7673"),
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
				"ffbfdcc705" + // locktime
				"0c646973747269627574696f6e"), // reference data
			hash:        mustDecodeHash("9ee3766feb8c66328b2f1a06b3a54cd3b4f02294f92f9b208cd99289d4b2898d"),
			witnessHash: mustDecodeHash("8e60cb00568facf7424a5779c86f631190a4882091ef474b2b1f1c047edab435"),
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
		{0, SigHashAll, "28591abea0b1f0ca4d1c4b141d623fd9cf9d1adb8c720b3863ed83ed75c4f05f"},
		{0, SigHashSingle, "2d8131ab90c895d831663d1424108c9438ed55b221b7bf210c6f7133214e97b5"},
		{0, SigHashNone, "57d254e0234cc0a0cb98a65ba680cf7357da3ab6022c38b50ae7e9e23027d87a"},
		{0, SigHashAll | SigHashAnyOneCanPay, "f2f855d28c4e9e25a04ae8e8a0920054870bf82caf818c3db2f2ae2eea4e40d9"},
		{0, SigHashSingle | SigHashAnyOneCanPay, "bad69b7690f9a963f53708475364f1fc105b2b59f931817a4ba3ee42b611f0af"},
		{0, SigHashNone | SigHashAnyOneCanPay, "5d6d24131cbddb05aaa2a219e540bc53381ca35c6236eeb7c9f101dee3f20827"},

		{1, SigHashAll, "33e33185a942f0564222e22c7bae6d6aeed816d9ea0ca5ede2e747dca2f6aec2"},
		{1, SigHashSingle, "ccfebd0050ac71a7dcdb95b539f87eea956e9fcf0fe36e46787b77f311682452"},
		{1, SigHashNone, "ff8ef1a5d5a64c331615c713eb85f2554924efc6a1cf9c5bca3400d58a950a9c"},
		{1, SigHashAll | SigHashAnyOneCanPay, "3f517862f8da9cfa9cb3b8a8aab0cc00567543e6863e5e463b5c6a3879b9eee5"},
		{1, SigHashSingle | SigHashAnyOneCanPay, "4f444febd0804da68dc61f57373c59f6338c76fe8e2fc9904cd48090a1592693"},
		{1, SigHashNone | SigHashAnyOneCanPay, "1e9f7de7e630db354690de60bfa88b03cfa06f7f48e6d745d83fd4f1a8185754"},
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
