package chainkd

import (
	"bytes"
	"reflect"
	"testing"

	"encoding/hex"
	"encoding/json"
)

func TestMarshalingFuncs(t *testing.T) {
	xprv, err := NewXPrv(nil)
	if err != nil {
		t.Fatal(err)
	}

	want := make([]byte, hex.EncodedLen(len(xprv.Bytes())))
	hex.Encode(want, xprv.Bytes())

	got, err := json.Marshal(xprv)
	if err != nil {
		t.Fatal(err)
	}
	// First and last bytes are "
	if !reflect.DeepEqual(want, got[1:len(got)-1]) {
		t.Errorf("marshaling error: want = %+v, got = %+v", want, got)
	}

	secXprv := new(XPrv)
	err = json.Unmarshal(got, &secXprv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(xprv[:], secXprv[:]) {
		t.Errorf("unmarshaling error: want = %+v, got = %+v", xprv, secXprv)
	}
}
