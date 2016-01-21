package asset_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	. "chain/api/asset"
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

func TestIssued(t *testing.T) {
	asset := [32]byte{255}
	outs := []*bc.TxOutput{
		{AssetAmount: bc.AssetAmount{AssetID: asset, Amount: 2}},
		{AssetAmount: bc.AssetAmount{AssetID: asset, Amount: 3}},
	}

	gotAsset, gotAmt := Issued(outs)
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
