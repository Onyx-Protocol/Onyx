package ca

import "chain/crypto/ed25519/ecmath"

// ValueRangeProof is a confidential value range proof.
type ValueRangeProof struct {
	nbits, exp uint8
	vmin       uint64
	digits     []PointPair
	brs        *BorromeanRingSignature
}

// CreateValueRangeProof creates a confidential value range proof.
func CreateValueRangeProof(AC *AssetCommitment, VC *ValueCommitment, N uint8, value uint64, pt [][32]byte, f ecmath.Scalar, idek DataKey, vek ValueKey, msg []byte) *ValueRangeProof {
	if len(pt) != int(2*N-1) {
		panic("calling error")
	}
	switch N {
	case 8, 16, 32, 48, 64:
		// do nothing
	default:
		return nil // xxx or panic?
	}
	if value >= 1<<N {
		return nil // xxx or panic?
	}
	const base = 4
	var (
		vmin uint64
		exp  uint8
	)
	msghash := hash256("ChainCA.VRP", AC.Bytes(), VC.Bytes(), uint64le(uint64(N)), uint64le(0), uint64le(0), msg)
	pek := hash256("ChainCA.pek", msghash[:], idek, f[:])
	n := N / 2
	ct := make([][64]byte, len(pt))
	var seed []byte
	for i, pti := range pt {
		EncryptPacket(pek[:], seed, pti[:], ct[i][:])
	}
	b := make([]ecmath.Scalar, n)
	bsum := ecmath.Zero
	hasher := streamHash("ChainCA.VRP.b", msghash[:], f[:])
	for t := 0; t < int(n-1); t++ {
		var bt [64]byte
		hasher.Read(bt[:])
		b[t].Reduce(&bt)
		bsum.Add(&bsum, &b[t])
	}
	b[n-1].Sub(&f, &bsum)

	var (
		digits = make([]PointPair, n)        // digits[t].Point1 is what the spec calls D[t], digits[t].Point2 is B[t]
		P      = make([][][]ecmath.Point, n) // P[t][i][0] is what the spec calls P[t,i], P[t][i][1] is Q[t,i]
		j      = make([]uint64, n)
	)

	baseToTheT := uint64(1)
	for t := uint(0); t < uint(n); t++ {
		digit := value & (0x03 << (1 << t))
		var digitScalar ecmath.Scalar
		digitScalar.SetUint64(digit)

		digits[t].Point1.ScMulAdd(&VC.Point1, &digitScalar, &b[t]) // D[t] = digit[t]·H + b[t]·G

		digits[t].Point2.ScMul(&VC.Point2, &digitScalar) // B[t] = digit[t]·C
		var T ecmath.Point
		T.ScMul(&J, &b[t])
		digits[t].Point2.Add(&digits[t].Point2, &T) // B[t] = digit[t]·C + b[t]·J

		j[t] = digit >> (2 * t)

		P[t] = make([][]ecmath.Point, base)

		for i := uint64(0); i < base; i++ {
			P[t][i] = []ecmath.Point{digits[t].Point1, digits[t].Point2}
			if i > 0 {
				iBaseToTheT := i * baseToTheT
				var iScalar ecmath.Scalar
				iScalar.SetUint64(iBaseToTheT)
				var T ecmath.Point
				T.ScMul(&VC.Point1, &iScalar)   // T = i·(base^t)·H
				P[t][i][0].Sub(&P[t][i][0], &T) // P[t,i] = D[t] - i·(base^t)·H
				T.ScMul(&VC.Point2, &iScalar)   // T = i·(base^t)·C
				P[t][i][1].Sub(&P[t][i][1], &T) // Q[t,i] = B[t] - i·(base^t)·C
			}
		}
		baseToTheT *= base
	}

	var fn []ecmath.Scalar
	for i := uint8(0); i < n; i++ {
		fn = append(fn, f)
	}

	var r [][32]byte // xxx r is ct reinterpreted

	brs := CreateBorromeanRingSignature(msghash[:], []ecmath.Point{G, J}, P, fn, j, r)
	return &ValueRangeProof{
		nbits:  N,
		exp:    exp,
		vmin:   vmin,
		digits: digits,
		brs:    brs,
	}
}
