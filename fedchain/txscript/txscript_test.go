package txscript

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestPkScriptAddr(t *testing.T) {
	cases := []struct {
		script string
		want   string
	}{
		{
			script: "a914a994a46855d8f4442b3a6db863628cc020537f4087",
			want:   "3H9gBofbYu4uQXwfMVcFiWjQHXf6vmnVGB",
		},
	}

	for _, c := range cases {
		h, err := hex.DecodeString(c.script)
		if err != nil {
			t.Fatal(err)
		}
		got, err := PkScriptAddr(h)
		if err != nil {
			t.Error("unexptected error", err)
		}
		if got.String() != c.want {
			t.Errorf("got pkScriptAddr(%s) = %v want %v", c.script, got, c.want)
		}
	}
}

func TestPkScriptToAssetID(t *testing.T) {
	cases := []struct {
		pkScript string
		want     string
	}{{
		pkScript: "a91468eab7b0cd1fd188e2abe1c1133d01689e0c10b587",
		want:     "AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo",
	}, {
		pkScript: "a91416dc6aa5e8aac191441e885c6ff17e111872b9e387",
		want:     "AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx",
	}, {
		pkScript: "a914b1cadb9ab9394ccc45ca4e4151e1b64970757e4487",
		want:     "ANzXFzvPWhubG1TfMDXskiAyHt5i8hREPn",
	}, {
		pkScript: "a914efea41476a7567044068760c0dd970f9bbd2d62087",
		want:     "AZ1nPNeoKjhb71bcNGAUDsFNnWWqg2kCno",
	}}
	for _, c := range cases {
		pkH, _ := hex.DecodeString(c.pkScript)
		hash := PkScriptToAssetID(pkH)

		if hash.String() != c.want {
			t.Errorf("got pkScriptToAssetID(%v) = %v want %v", c.pkScript, hash.String(), c.want)
		}
	}
}

// Taken from PAPI
func TestRedeemToPkScript(t *testing.T) {
	redeem := []byte{
		82, 65, 4, 2, 83, 21, 116, 23, 208, 223, 22, 63, 33, 52, 55, 175, 75,
		119, 114, 250, 19, 22, 177, 255, 206, 20, 137, 199, 197, 174, 244, 194,
		15, 245, 81, 94, 80, 76, 230, 243, 156, 11, 161, 17, 245, 68, 250, 134,
		98, 63, 123, 206, 106, 17, 129, 179, 210, 5, 155, 242, 97, 194, 119,
		175, 122, 32, 45, 65, 4, 219, 47, 252, 31, 82, 125, 34, 225, 107, 200,
		88, 45, 78, 46, 221, 232, 119, 33, 245, 22, 107, 5, 210, 37, 38, 160,
		107, 38, 218, 198, 70, 140, 97, 52, 204, 27, 97, 252, 237, 156, 154,
		175, 86, 193, 177, 245, 210, 222, 244, 235, 8, 179, 15, 187, 126, 249,
		192, 138, 143, 251, 198, 230, 98, 172, 82, 174,
	}

	want := []byte{
		169, 20, 10, 63, 117, 193, 26, 249, 104, 211, 169, 228, 39, 135, 197,
		179, 65, 183, 169, 3, 163, 165, 135,
	}

	got, err := RedeemToPkScript(redeem)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if bytes.Compare(got, want) != 0 {
		t.Errorf("got pkscript = %x want %x", got, want)
	}
}
