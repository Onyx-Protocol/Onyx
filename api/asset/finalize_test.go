package asset

import (
	"bytes"
	"encoding/hex"
	"testing"

	"chain/errors"
	"chain/fedchain/bc"
)

func mustDecodeHex(data string) []byte {
	h, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}
	return h
}

func TestAssembleSignatures(t *testing.T) {
	outscript := mustDecodeHex("a9140ac9c982fd389181752e5a414045dd424a10754b87")
	unsigned := &bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Index: bc.InvalidOutputIndex}}},
		Outputs: []*bc.TxOutput{{AssetAmount: bc.AssetAmount{AssetID: [32]byte{255}, Amount: 5}, Script: outscript}},
	}
	sigHash, _ := bc.ParseHash("78e437f627019fc270bbe9ed309291d0a5f6bf98bfae0f750538ba56646f7327")

	tpl := &TxTemplate{
		Unsigned: unsigned,
		Inputs: []*Input{{
			SignatureData: sigHash,
			Sigs: []*Signature{{
				XPub:           "xpub661MyMwAqRbcGiDB8FQvHnDAZyaGUyzm3qN1Q3NDJz1PgAWCfyi9WRCS7Z9HyM5QNEh45fMyoaBMqjfoWPdnktcN8chJYB57D2Y7QtNmadr",
				DerivationPath: []uint32{0, 0, 0, 0},
				DER:            mustDecodeHex("3044022004da5732f6c988b9e2882f5ca4f569b9525d313940e0372d6a84fef73be78f8f02204656916481dc573d771ec42923a8f5af31ae634241a4cb30ea5b359363cf064d"),
			}},
			RedeemScript: []byte{0},
		}},
		OutRecvs: []Receiver{nil}, // pays to external party
	}

	tx, err := assembleSignatures(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}

	want := "a83ee0b537eead1f68c9d77657ebb2407833641cc44c85e2c7a43b8a9d50c337"
	if tx.Hash.String() != want {
		t.Errorf("got tx hash = %v want %v", tx.Hash.String(), want)
	}
}

func TestIssued(t *testing.T) {
	asset := [32]byte{255}
	outs := []*bc.TxOutput{
		{AssetAmount: bc.AssetAmount{AssetID: asset, Amount: 2}},
		{AssetAmount: bc.AssetAmount{AssetID: asset, Amount: 3}},
	}

	gotAsset, gotAmt := issued(outs)
	if !bytes.Equal(gotAsset[:], asset[:]) {
		t.Errorf("got asset = %q want %q", gotAsset.String(), bc.AssetID(asset).String())
	}

	if gotAmt != 5 {
		t.Errorf("got amt = %d want %d", gotAmt, 5)
	}
}

func withStack(err error) string {
	s := err.Error()
	for _, frame := range errors.Stack(err) {
		s += "\n" + frame.String()
	}
	return s
}
