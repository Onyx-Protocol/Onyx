package bc

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/txscript"

	"chain/fedchain/script"
)

func TestComputeAssetID(t *testing.T) {
	issuanceScript := script.Script{txscript.OP_1}
	genesisHash := decodeHash256("e5f90ce43c924a0e57284ad1ff93618c19c997e53b3c4b3d4d903f4c5d6f50dd")
	assetID := ComputeAssetID(issuanceScript, genesisHash)

	want, _ := hex.DecodeString("1ccfc833562fde9678bdade004f594111d688ed9")
	if !bytes.Equal(assetID[:], want) {
		t.Errorf("asset id = %x want %x", assetID[:], want)
	}
}

func TestComputeIssuanceID(t *testing.T) {
	got := ComputeIssuanceID(Outpoint{})
	want, _ := hex.DecodeString("d33b70b905f59fac92d69d0978917524660a5f13")
	if !bytes.Equal(got[:], want) {
		t.Errorf("asset id = %x want %x", got[:], want)
	}
}
