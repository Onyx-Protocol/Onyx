package ca

import "chain/crypto/ed25519/ecmath"

// G is a base point
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

// TBD: Precomputed reminder generators: GR[i] = G - Sum[G[j], j<i]; GR[0] = G
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
	Gbuf := G.Encode()
	return pointHash("ChainCA.J", Gbuf[:])
}

func makeGiPure(i byte) ecmath.Point {
	p, _ := makeGi(i)
	return p
}

func makeGi(i byte) (P ecmath.Point, ctr uint64) {
	Gbuf := G.Encode()
	for ctr = uint64(0); true; ctr++ {
		// 1. Calculate `SHA3-256(i || Encode(G) || counter64le)`
		h := sha3_256([]byte{i}, Gbuf[:], uint64le(ctr))

		// 2. Decode the resulting hash as a point `P` on the elliptic curve.
		_, ok := P.Decode(h)

		if !ok {
			continue
		}

		// 3. Calculate point `G[i] = 8*P` (8 is a cofactor in edwards25519) which belongs to a subgroup of `G` with order `L`.
		cofactor := ecmath.Scalar{8}
		P.ScMul(&P, &cofactor)

		break
	}
	return
}
