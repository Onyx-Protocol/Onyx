package ca

import "chain/crypto/ed25519/ecmath"

var powersOf10, invPowersOf10 [20]ecmath.Scalar

// 10^1
var ten = ecmath.Scalar{10}

// 10^(-1) mod (subgroup order): 723700557733226221397318656304299424085711635937990760600195093828545425099
var tenInv = ecmath.Scalar{
	0xcb, 0x2e, 0xb2, 0x6f, 0x4f, 0x70, 0x9b, 0xd5,
	0x7b, 0x5c, 0xb2, 0x76, 0xc9, 0xe5, 0xaf, 0x9b,
	0x99, 0x99, 0x99, 0x99, 0x99, 0x99, 0x99, 0x99,
	0x99, 0x99, 0x99, 0x99, 0x99, 0x99, 0x99, 0x01,
}

func init() {
	powersOf10[0] = ecmath.One
	invPowersOf10[0] = ecmath.One

	powersOf10[1] = ten
	invPowersOf10[1] = tenInv

	for i := 2; i < 20; i++ {
		powersOf10[i].Mul(&powersOf10[i-1], &ten)
		invPowersOf10[i].Mul(&invPowersOf10[i-1], &tenInv)
	}
}

// -19 <= n <= 19
func powerOf10(n int) *ecmath.Scalar {
	if n < 0 {
		return &invPowersOf10[-n]
	}
	return &powersOf10[n]
}
