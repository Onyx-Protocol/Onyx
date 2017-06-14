package ca

import (
	"chain/crypto/ed25519/ecmath"
)

type OlegZKP struct {
	e ecmath.Scalar
	s [][]ecmath.Scalar
}

type OlegZKPFunc func([]ecmath.Scalar) ecmath.Point

func CreateOlegZKP(msg []byte, x []ecmath.Scalar, f [][]OlegZKPFunc, F [][]ecmath.Point, iHat uint64) *OlegZKP {
	l := uint64(len(x))
	msghash := ozkpMsgHash(F, l, msg)
	return createOlegZKP(msghash[:], x, f, F, iHat, 0)
}

func createOlegZKP(msghash []byte, x []ecmath.Scalar, f [][]OlegZKPFunc, F [][]ecmath.Point, iHat, counter uint64) *OlegZKP {
	n := uint64(len(F))
	l := uint64(len(x))

	var (
		S    = make([][]ecmath.Scalar, n-1)
		r    = make([]ecmath.Scalar, l)
		mask = make([]byte, l)
	)

	stream := streamHash("ChainCA.OZKP.rand", uint64le(counter), msghash)
	for _, xi := range x {
		stream.Write(xi[:])
	}
	for i := uint64(0); i < n-1; i++ {
		S[i] = make([]ecmath.Scalar, l)
		for j := uint64(0); j < l; j++ {
			stream.Read(S[i][j][:])
		}
	}
	for i := uint64(0); i < l; i++ {
		var r64 [64]byte
		stream.Read(r64[:])
		r[i].Reduce(&r64)
	}
	stream.Read(mask[:])

	e := make([]ecmath.Scalar, n)

	iPrime := (iHat + 1) % n
	w := make([]byte, l)
	for k := uint64(0); k < l; k++ {
		w[k] = mask[k] & 0xf0
	}
	e[iPrime] = ozkpNextE(msghash, iPrime, f[iHat], r, w, nil, nil)

	s := make([][]ecmath.Scalar, n)
	z := make([]ecmath.Scalar, l)
	for step := uint64(1); step < n; step++ {
		i := (iHat + step) % n
		s[i] = make([]ecmath.Scalar, l)
		for k := uint64(0); k < l; k++ {
			s[i][k] = S[step-1][k]
			z[k] = s[i][k]
			z[k][31] &= 0x0f
			w[k] = s[i][k][31] & 0xf0
		}
		iPrime := (i + 1) % n
		e[iPrime] = ozkpNextE(msghash, iPrime, f[i], z, w, &e[i], F[i])
	}

	s[iHat] = make([]ecmath.Scalar, l)
	for k := uint64(0); k < l; k++ {
		z[k].MulAdd(&x[k], &e[iHat], &r[k])
		if z[k][31]&0xf0 != 0 {
			return createOlegZKP(msghash, x, f, F, iHat, counter+1)
		}
		s[iHat][k] = z[k]
		s[iHat][k][31] &= 0x0f
		s[iHat][k][31] |= (mask[k] & 0xf0)
	}

	return &OlegZKP{e: e[0], s: s}
}

func ozkpMsgHash(F [][]ecmath.Point, l uint64, msg []byte) [32]byte {
	n := uint64(len(F))
	m := uint64(len(F[0]))
	hasher := hasher256("ChainCA.OZKP.msg", uint64le(n), uint64le(m), uint64le(l))
	for _, Fi := range F {
		for _, Fij := range Fi {
			hasher.WriteItem(Fij.Bytes())
		}
	}
	hasher.WriteItem(msg)
	var result [32]byte
	hasher.Sum(result[:0])
	return result
}

func ozkpEHash(msghash []byte, i uint64, R []ecmath.Point, w []byte) ecmath.Scalar {
	hasher := scalarHasher("ChainCA.OZKP.e", msghash, uint64le(i))
	for _, Ri := range R {
		hasher.WriteItem(Ri.Bytes())
	}
	hasher.WriteItem(w)
	return scalarHasherFinalize(hasher)
}

// Note: e == nil means R[j] = f(r[0], ..., r[n-1])
//       e != nil means R[j] = f(r[0], ..., r[n-1]) - e*F[j]
func ozkpNextE(msghash []byte, i uint64, f []OlegZKPFunc, r []ecmath.Scalar, w []byte, e *ecmath.Scalar, F []ecmath.Point) ecmath.Scalar {
	m := len(f)
	R := make([]ecmath.Point, m)
	for j := 0; j < m; j++ {
		R[j] = f[j](r)
		if e != nil {
			var R2 ecmath.Point
			R2.ScMul(&F[j], e)
			R[j].Sub(&R[j], &R2)
		}
	}
	return ozkpEHash(msghash, i, R, w)
}

func (ozkp *OlegZKP) Validate(msg []byte, f [][]OlegZKPFunc, F [][]ecmath.Point) bool {
	n := uint64(len(F))
	l := uint64(len(ozkp.s[0]))
	msghash := ozkpMsgHash(F, l, msg)
	e := make([]ecmath.Scalar, n+1)
	e[0] = ozkp.e
	for i := uint64(0); i < n; i++ {
		iPrime := (i + 1) % n
		z := make([]ecmath.Scalar, l)
		w := make([]byte, l)
		for k := uint64(0); k < l; k++ {
			z[k] = ozkp.s[i][k]
			z[k][31] &= 0x0f
			w[k] = ozkp.s[i][k][31] & 0xf0
		}
		e[i+1] = ozkpNextE(msghash[:], iPrime, f[i], z, w, &e[i], F[i])
	}
	return e[0] == e[n]
}
