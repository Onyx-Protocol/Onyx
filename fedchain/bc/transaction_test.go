package bc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/txscript"

	"chain/fedchain/script"
)

func TestTransaction(t *testing.T) {
	issuanceScript := script.Script{txscript.OP_1}
	genesisHash := mustDecodeHash("dd506f5d4c3f904d3d4b3c3be597c9198c6193ffd14a28570e4a923ce40cf9e5")

	cases := []struct {
		tx   Tx
		hex  string
		hash [32]byte
	}{
		{
			tx: Tx{
				Version:  CurrentTransactionVersion,
				Inputs:   nil,
				Outputs:  nil,
				LockTime: 0,
				Metadata: nil,
			},
			hex:  "000000010000000000000000000000",
			hash: mustDecodeHash("6ded6af33b14c1d4745cea6965b3483f642057c057724d6ea2df05fc78bc5b4d"),
		},
		{
			tx: Tx{
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
						AssetID:  AssetID{},
						Value:    1000000000000,
						Script:   script.Script{txscript.OP_1},
						Metadata: []byte("output"),
					},
				},
				LockTime: 0,
				Metadata: []byte("issuance"),
			},
			hex:  "0000000101dd506f5d4c3f904d3d4b3c3be597c9198c6193ffd14a28570e4a923ce40cf9e5ffffffff090869737375616e636505696e707574010000000000000000000000000000000000000000000000000000000000000000000000e8d4a510000151066f757470757400000000000000000869737375616e6365",
			hash: mustDecodeHash("a481c6bfd9f2df7b786d61f979fa9c21cd25840c16abf6b1e3bbdd883b06a81d"),
		},
		{
			tx: Tx{
				Version: CurrentTransactionVersion,
				Inputs: []*TxInput{
					{
						Previous: Outpoint{
							Hash:  mustDecodeHash("92322db99e8b9e9f1df601cc9d22c5b056ad5189a50fbdc1d8915de26f5f38dd"),
							Index: 0,
						},
						SignatureScript: nil,
						Metadata:        []byte("input"),
					},
				},
				Outputs: []*TxOutput{
					{
						AssetID:  ComputeAssetID(issuanceScript, genesisHash),
						Value:    600000000000,
						Script:   script.Script{txscript.OP_1},
						Metadata: nil,
					},
					{
						AssetID:  ComputeAssetID(issuanceScript, genesisHash),
						Value:    400000000000,
						Script:   script.Script{txscript.OP_2},
						Metadata: nil,
					},
				},
				LockTime: 1492590591,
				Metadata: []byte("distribution"),
			},
			hex:  "000000010192322db99e8b9e9f1df601cc9d22c5b056ad5189a50fbdc1d8915de26f5f38dd000000000005696e70757402a0f16ffd5618342611dd52589cad51f93e40cb9c54ab2e18c3169ca2e511533f0000008bb2c97000015100a0f16ffd5618342611dd52589cad51f93e40cb9c54ab2e18c3169ca2e511533f0000005d21dba0000152000000000058f71fff0c646973747269627574696f6e",
			hash: mustDecodeHash("4a8f5cca42fcc4321cf85cb6134b01c61ddbc9fcab5d309e73bb5e7e3781960b"),
		},
	}

	for _, test := range cases {
		t.Logf("metadata %q", test.tx.Metadata)

		got := serialize(t, &test.tx)
		want, _ := hex.DecodeString(test.hex)
		if !bytes.Equal(got, want) {
			t.Errorf("bytes = %x want %x", got, want)
		}
		hash := test.tx.Hash()
		if !bytes.Equal(hash[:], test.hash[:]) {
			t.Errorf("hash = %x want %x", hash, test.hash)
		}

		tx1 := new(Tx)
		err := tx1.UnmarshalText([]byte(test.hex))
		if err != nil {
			t.Errorf("unexpected err %v", err)
		}
		if !reflect.DeepEqual(*tx1, test.tx) {
			t.Errorf("tx1 = %v want %v", *tx1, test.tx)
		}
	}
}

func TestIsIssuance(t *testing.T) {
	tx := Tx{
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
				AssetID:  AssetID{},
				Value:    1000000000000,
				Script:   script.Script{txscript.OP_1},
				Metadata: []byte("output"),
			},
		},
		LockTime: 0,
		Metadata: []byte("issuance"),
	}

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
