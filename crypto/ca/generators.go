package ca

import (
	"chain/crypto/ed25519/ecmath"
)

var G = makeG()
var J = makeJ()

var Gi = []ecmath.Point{
	makeGiPure(0), makeGiPure(1), makeGiPure(2), makeGiPure(3),
	makeGiPure(4), makeGiPure(5), makeGiPure(6), makeGiPure(7),
	makeGiPure(8), makeGiPure(9), makeGiPure(10), makeGiPure(11),
	makeGiPure(12), makeGiPure(13), makeGiPure(14), makeGiPure(15),
	makeGiPure(16), makeGiPure(17), makeGiPure(18), makeGiPure(19),
	makeGiPure(20), makeGiPure(21), makeGiPure(22), makeGiPure(23),
	makeGiPure(24), makeGiPure(25), makeGiPure(26), makeGiPure(27),
	makeGiPure(28), makeGiPure(29), makeGiPure(30),
}

// Precomputed reminder generators: GR[i] = G - Sum[G[j], j<i]; GR[0] = G
var GRi = []ecmath.Point{
	// TBD: these are not correctly computed
	makeGiPure(0), makeGiPure(1), makeGiPure(2), makeGiPure(3),
	makeGiPure(4), makeGiPure(5), makeGiPure(6), makeGiPure(7),
	makeGiPure(8), makeGiPure(9), makeGiPure(10), makeGiPure(11),
	makeGiPure(12), makeGiPure(13), makeGiPure(14), makeGiPure(15),
	makeGiPure(16), makeGiPure(17), makeGiPure(18), makeGiPure(19),
	makeGiPure(20), makeGiPure(21), makeGiPure(22), makeGiPure(23),
	makeGiPure(24), makeGiPure(25), makeGiPure(26), makeGiPure(27),
	makeGiPure(28), makeGiPure(29), makeGiPure(30),
}

func makeG() (r ecmath.Point) {
	r.ScMulBase(&ecmath.One)
	return
}

func makeJ() (j ecmath.Point) {
	// Decode the point from SHA3(G)
	h := hash256(G.Encode())
	err := j.fromBytes(&h)
	if err != nil {
		panic("failed to decode secondary generator")
	}
	// Calculate point `J = 8*J` (8 is a cofactor in edwards25519) which belongs to a subgroup of `G` with order `L`.
	j.mul(&cofactor)
	return
}

func makeGiPure(i byte) Point {
	p, _ := makeGi(i)
	return p
}

func makeGi(i byte) (P Point, ctr uint64) {
	Gbytes := G.bytes()
	for ctr = uint64(0); true; ctr++ {
		// 1. Calculate `SHA3-256(i || Encode(G) || counter64le)`
		h := hash256([]byte{i}, Gbytes, uint64le(ctr))

		// 2. Decode the resulting hash as a point `P` on the elliptic curve.
		err := P.fromBytes(&h)

		if err != nil {
			continue
		}

		// 3. Calculate point `G[i] = 8*P` (8 is a cofactor in edwards25519) which belongs to a subgroup of `G` with order `L`.
		cofactor := ecmath.Scalar{8}
		P.mul(&cofactor)

		break
	}
	return
}
