package bc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/btcsuite/btcd/txscript"

	"chain/fedchain/script"
)

func TestTransaction(t *testing.T) {
	issuanceScript := script.Script{txscript.OP_1}
	genesisHash := decodeHash256("e5f90ce43c924a0e57284ad1ff93618c19c997e53b3c4b3d4d903f4c5d6f50dd")

	cases := []struct {
		tx  Tx
		hex string
		id  string
	}{
		{
			tx: Tx{
				Version:  CurrentTransactionVersion,
				Inputs:   nil,
				Outputs:  nil,
				LockTime: 0,
				Metadata: "",
			},
			hex: "010000000000000000000000000000",
			id:  "5c453d897a609c8cc46958fbd436a52bd3a2ce8f8829aa8a14d500fa44d63300",
		},
		{
			tx: Tx{
				Version: CurrentTransactionVersion,
				Inputs: []TxInput{
					{
						Previous: Outpoint{
							Hash:  decodeHash256("e5f90ce43c924a0e57284ad1ff93618c19c997e53b3c4b3d4d903f4c5d6f50dd"),
							Index: InvalidOutputIndex,
						},
						// "PUSHDATA 'issuance'"
						SignatureScript: []byte{txscript.OP_DATA_8, 0x69, 0x73, 0x73, 0x75, 0x61, 0x6e, 0x63, 0x65},
						Metadata:        []byte("input"),
					},
				},
				Outputs: []TxOutput{
					{
						AssetID:  AssetID{},
						Value:    1000000000000,
						Script:   script.Script{txscript.OP_1},
						Metadata: []byte("output"),
					},
				},
				LockTime: 0,
				Metadata: "issuance",
			},
			hex: "0100000001dd506f5d4c3f904d3d4b3c3be597c9198c6193ffd14a28570e4a923ce40cf9e5ffffffff090869737375616e636505696e7075740100000000000000000000000000000000000000000000000000000000000000000010a5d4e80000000151066f757470757400000000000000000869737375616e6365",
			id:  "24c7a3db6d108dda4ae880bc5ea6346477a6c86bd1e54a12c592c61d5dbe6741",
		},
		{
			tx: Tx{
				Version: CurrentTransactionVersion,
				Inputs: []TxInput{
					{
						Previous: Outpoint{
							Hash:  decodeHash256("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"),
							Index: 0,
						},
						SignatureScript: script.Script{},
						Metadata:        []byte("input"),
					},
				},
				Outputs: []TxOutput{
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
				Metadata: "distribution",
			},
			hex: "010000000192322db99e8b9e9f1df601cc9d22c5b056ad5189a50fbdc1d8915de26f5f38dd000000000005696e70757402a0f16ffd5618342611dd52589cad51f93e40cb9c54ab2e18c3169ca2e511533f0070c9b28b000000015100a0f16ffd5618342611dd52589cad51f93e40cb9c54ab2e18c3169ca2e511533f00a0db215d000000015200ff1ff758000000000c646973747269627574696f6e",
			id:  "7f0a1fca0143c46d1f457949d1ae3da426ac9f67c546971dca2e55c5b3e2ee2d",
		},
	}

	for _, test := range cases {
		t.Logf("metadata %q", test.tx.Metadata)

		got := serialize(t, &test.tx)
		want, _ := hex.DecodeString(test.hex)
		if !bytes.Equal(got, want) {
			t.Errorf("bytes = %x want %x", got, want)
		}

		h := test.tx.Hash()
		if got := ID(h[:]); got != test.id {
			t.Errorf("id = %s want %s", got, test.id)
		}
	}
}

func TestIsIssuance(t *testing.T) {
	tx := Tx{
		Version: CurrentTransactionVersion,
		Inputs: []TxInput{
			{
				Previous: Outpoint{
					Hash:  decodeHash256("e5f90ce43c924a0e57284ad1ff93618c19c997e53b3c4b3d4d903f4c5d6f50dd"),
					Index: InvalidOutputIndex,
				},
				// "PUSHDATA 'issuance'"
				SignatureScript: []byte{txscript.OP_DATA_8, 0x69, 0x73, 0x73, 0x75, 0x61, 0x6e, 0x63, 0x65},
				Metadata:        []byte("input"),
			},
		},
		Outputs: []TxOutput{
			{
				AssetID:  AssetID{},
				Value:    1000000000000,
				Script:   script.Script{txscript.OP_1},
				Metadata: []byte("output"),
			},
		},
		LockTime: 0,
		Metadata: "issuance",
	}

	if g := tx.Inputs[0].IsIssuance(); !g {
		t.Errorf("input IsIssuance() = %v want true", g)
	}
	if g := tx.IsIssuance(); !g {
		t.Errorf("tx IsIssuance() = %v want true", g)
	}
}

func TestEmptyOutpoint(t *testing.T) {
	o := Outpoint{
		Hash:  [32]byte{0},
		Index: 0,
	}

	if o.String() != "0000000000000000000000000000000000000000000000000000000000000000:0" {
		t.Errorf("Empty outpoint has incorrect string representation '%v'", o.String())
	}
}

func TestIssuanceOutpoint(t *testing.T) {
	o := Outpoint{
		Hash:  decodeHash256("292cdb9fd93f3f78c6b7a24e92b805a8dcf363787fe9c433259b8bc4227dc2fb"),
		Index: 0xffffffff,
	}
	if o.String() != "292cdb9fd93f3f78c6b7a24e92b805a8dcf363787fe9c433259b8bc4227dc2fb:4294967295" {
		t.Errorf("Issuance outpoint has incorrect string representation '%v'", o.String())
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
