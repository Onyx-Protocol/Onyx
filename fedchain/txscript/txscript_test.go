package txscript

import (
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
