package bc

import (
	"bytes"
	"chain/types"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func TestMarshalBlock(t *testing.T) {
	b := &Block{
		BlockHeader: BlockHeader{
			Version: 1,
			Height:  1,
		},

		Transactions: []*Tx{
			NewTx(TxData{
				Version: 1,
				Outputs: []*TxOutput{
					NewTxOutput(types.AssetID{}, 1, nil, nil),
				},
			}),
		}}

	got, err := json.Marshal(b)
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	// Include start and end quote marks because json.Marshal adds them
	// to the result of Block.MarshalText.
	wantHex := ("\"03" + // serialization flags
		"01" + // version
		"01" + // block height
		"0000000000000000000000000000000000000000000000000000000000000000" + // prev block hash
		"00" + // timestamp
		"41" + // commitment extensible field length
		"0000000000000000000000000000000000000000000000000000000000000000" + // tx merkle root
		"0000000000000000000000000000000000000000000000000000000000000000" + // assets merkle root
		"00" + // consensus program
		"01" + // witness extensible string length
		"00" + // witness number of witness args
		"01" + // num transactions
		"07" + // tx 0, serialization flags
		"01" + // tx 0, tx version
		"02" + // tx 0, common fields extensible length string
		"00" + // tx 0, common fields mintime
		"00" + // tx 0, common fields maxtime
		"00" + // tx 0, common witness extensible string length
		"00" + // tx 0, inputs count
		"01" + // tx 0, outputs count
		"01" + // tx 0 output 0, asset version
		"23" + // tx 0, output 0, output commitment length
		"0000000000000000000000000000000000000000000000000000000000000000" + // tx 0, output 0 commitment, asset id
		"01" + // tx 0, output 0 commitment, amount
		"01" + // tx 0, output 0 commitment vm version
		"00" + // tx 0, output 0 control program
		"00" + // tx 0, output 0 reference data
		"00" + // tx 0, output 0 output witness
		"00\"") // tx 0 reference data

	if !bytes.Equal(got, []byte(wantHex)) {
		t.Errorf("marshaled block bytes = %s want %s", got, []byte(wantHex))
	}

	var c Block
	err = json.Unmarshal(got, &c)
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	if !reflect.DeepEqual(*b, c) {
		t.Errorf("expected marshaled/unmarshaled block to be:\n%sgot:\n%s", spew.Sdump(*b), spew.Sdump(c))
	}

	got[7] = 'q'
	err = json.Unmarshal(got, &c)
	if err == nil {
		t.Error("unmarshaled corrupted JSON ok, wanted error")
	}
}

func TestEmptyBlock(t *testing.T) {
	block := Block{
		BlockHeader: BlockHeader{
			Version: NewBlockVersion,
			Height:  1,
		},
	}

	got := serialize(t, &block)
	wantHex := ("03" + // serialization flags
		"01" + // version
		"01" + // block height
		"0000000000000000000000000000000000000000000000000000000000000000" + // prev block hash
		"00" + // timestamp
		"41" + // commitment extensible field length
		"0000000000000000000000000000000000000000000000000000000000000000" + // transactions merkle root
		"0000000000000000000000000000000000000000000000000000000000000000" + // assets merkle root
		"00" + // consensus program
		"01" + // witness extensible string length
		"00" + // witness number of witness args
		"00") // num transactions
	want, _ := hex.DecodeString(wantHex)
	if !bytes.Equal(got, want) {
		t.Errorf("empty block bytes = %x want %x", got, want)
	}

	got = serialize(t, &block.BlockHeader)
	wantHex = ("01" + // serialization flags
		"01" + // version
		"01" + // block height
		"0000000000000000000000000000000000000000000000000000000000000000" + // prev block hash
		"00" + // timestamp
		"41" + // commitment extensible field length
		"0000000000000000000000000000000000000000000000000000000000000000" + // transactions merkle root
		"0000000000000000000000000000000000000000000000000000000000000000" + // assets merkle root
		"00" + // consensus program
		"01" + // witness extensible string length
		"00") // witness number of witness args
	want, _ = hex.DecodeString(wantHex)
	if !bytes.Equal(got, want) {
		t.Errorf("empty block header bytes = %x want %x", got, want)
	}

	wantHash := mustDecodeHash("7508682af2b4770e327b26ad52809da99bd89d885b91d4fba44e93bd0ad1da2f")
	if h := block.Hash(); h != wantHash {
		t.Errorf("empty block has incorrect hash %s", h)
	}
	wantHash = mustDecodeHash("a48b8fc5a149250b68ee77606175c23d36d6933c178d5645b5b1d1e89e130207")
	if h := block.HashForSig(); h != wantHash {
		t.Errorf("empty block has incorrect sig hash %s", h)
	}

	wTime := time.Unix(0, 0).UTC()
	if got := block.Time(); got != wTime {
		t.Errorf("empty block time = %v want %v", got, wTime)
	}
}

func TestSmallBlock(t *testing.T) {
	block := Block{
		BlockHeader: BlockHeader{
			Version: NewBlockVersion,
			Height:  1,
		},
		Transactions: []*Tx{NewTx(TxData{Version: CurrentTransactionVersion})},
	}

	got := serialize(t, &block)
	wantHex := ("03" + // serialization flags
		"01" + // version
		"01" + // block height
		"0000000000000000000000000000000000000000000000000000000000000000" + // prev block hash
		"00" + // timestamp
		"41" + // commitment extensible field length
		"0000000000000000000000000000000000000000000000000000000000000000" + // transactions merkle root
		"0000000000000000000000000000000000000000000000000000000000000000" + // assets merkle root
		"00" + // consensus program
		"01" + // witness extensible string length
		"00" + // witness num witness args
		"01" + // num transactions
		"070102000000000000") // transaction
	want, _ := hex.DecodeString(wantHex)
	if !bytes.Equal(got, want) {
		t.Errorf("small block bytes = %x want %x", got, want)
	}
}
