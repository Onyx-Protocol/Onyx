package ca

import (
	"encoding/hex"
	"fmt"
	"testing"

	"chain/crypto/ed25519/ecmath"
)

func TestRingSig(t *testing.T) {
	cases := []struct {
		basePoints []ecmath.Point
		privkeyHex []string // privkeyHex[0] is the real secret
	}{
		{
			basePoints: []ecmath.Point{G},
			privkeyHex: []string{"4f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da04"},
		},
		{
			basePoints: []ecmath.Point{G, J},
			privkeyHex: []string{"4f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da04"},
		},
		{
			basePoints: []ecmath.Point{G},
			privkeyHex: []string{
				"4f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da04",
				"59e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e04716601",
				"483262b2722ec4a6e967af08d0ed3d51f954e2b9cab2b51b47aca3d80a58aa0f",
			},
		},
		{
			basePoints: []ecmath.Point{G, J},
			privkeyHex: []string{
				"4f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da04",
				"59e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e04716601",
				"483262b2722ec4a6e967af08d0ed3d51f954e2b9cab2b51b47aca3d80a58aa0f",
			},
		},
	}

	msg := []byte("message")

	var p ecmath.Scalar

	for _, c := range cases {
		n := len(c.privkeyHex)
		m := len(c.basePoints)
		t.Run(fmt.Sprintf("n%dm%d", n, m), func(t *testing.T) {
			P := make([][]ecmath.Point, n)
			for i := 0; i < n; i++ {
				P[i] = make([]ecmath.Point, m)
				var p2 ecmath.Scalar
				hex.Decode(p2[:], []byte(c.privkeyHex[i]))
				for u := 0; u < m; u++ {
					P[i][u].ScMul(&c.basePoints[u], &p2)
				}
				if i == 0 {
					p = p2
				}
			}
			rs := CreateRingSignature(msg, c.basePoints, P, 0, p)
			if !rs.Validate(msg, c.basePoints, P) {
				t.Error("failed to validate ring signature")
			}
			if rs.Validate(msg[1:], c.basePoints, P) {
				t.Error("validated invalid ring signature")
			}
		})
	}
}
