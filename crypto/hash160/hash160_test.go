package hash160

import (
	"encoding/hex"
	"testing"
)

func TestSum(t *testing.T) {
	cases := []struct{ data, want string }{
		{"a", "994355199e516ff76c4fa4aab39337b9d84cf12b"},
	}

	for _, test := range cases {
		hash := Sum([]byte(test.data))
		got := hex.EncodeToString(hash[:])
		if got != test.want {
			t.Errorf("Sum(%q) = %s want %s", test.data, got, test.want)
		}
	}
}
