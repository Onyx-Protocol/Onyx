package hash256

import (
	"encoding/hex"
	"testing"
)

func TestSum(t *testing.T) {
	cases := []struct{ data, want string }{
		{"a", "bf5d3affb73efd2ec6c36ad3112dd933efed63c4e1cbffcfa88e2759c144f2d8"},
	}

	for _, test := range cases {
		hash := Sum([]byte(test.data))
		got := hex.EncodeToString(hash[:])
		if got != test.want {
			t.Errorf("Sum(%q) = %s want %s", test.data, got, test.want)
		}
	}
}
