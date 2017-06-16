package ca

import "chain/crypto/ed25519/ecmath"

// ValueRangeProof is a confidential value range proof.
type ValueRangeProof struct {
	nbits, exp uint8
	vmin       uint64
	digits     []PointPair
	brs        *BorromeanRingSignature
}

const base = 4

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
	vmin := uint64(0)
	exp := uint8(0)
	msghash := vrpMsgHash(AC, VC, N, 0, 0, msg)
	pek := hash256("ChainCA.pek", msghash[:], idek, f[:])
	n := N / 2

	buf := make([]byte, 0, 32*2*N)
	for _, pti := range pt {
		buf = append(buf, pti[:]...)
	}
	EncryptPacket(pek[:], nil, buf[:32*(2*N-1)], buf[:])
	ct := make([][32]byte, 2*N)
	for i := uint8(0); i < 2*N; i++ {
		copy(ct[i][:], buf[32*i:32*(i+1)])
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
		digits = make([]PointPair, n)        // digits[t][0] is what the spec calls D[t], digits[t][1] is B[t]
		P      = make([][][]ecmath.Point, n) // P[t][i][0] is what the spec calls P[t,i], P[t][i][1] is Q[t,i]
		j      = make([]uint64, n)
	)

	baseToTheT := uint64(1)
	for t := uint(0); t < uint(n); t++ {
		digit := value & (0x03 << (1 << t))
		d := (&ecmath.Scalar{}).SetUint64(digit)

		digits[t][0].ScMulAdd(&VC[0], d, &b[t]) // D[t] = digit[t]·H + b[t]·G

		digits[t][1].ScMul(&VC[1], d) // B[t] = digit[t]·C
		var T ecmath.Point
		T.ScMul(&J, &b[t])
		digits[t][1].Add(&digits[t][1], &T) // B[t] = digit[t]·C + b[t]·J

		j[t] = digit >> (2 * t)

		P[t] = make([][]ecmath.Point, base)

		for i := uint64(0); i < base; i++ {
			P[t][i] = make([]ecmath.Point, 2)
			copy(P[t][i][:], digits[t][:])
			if i > 0 {
				iBaseToTheT := i * baseToTheT
				var iScalar ecmath.Scalar
				iScalar.SetUint64(iBaseToTheT)
				var T ecmath.Point
				T.ScMul(&VC[0], &iScalar)       // T = i·(base^t)·H
				P[t][i][0].Sub(&P[t][i][0], &T) // P[t,i] = D[t] - i·(base^t)·H
				T.ScMul(&VC[1], &iScalar)       // T = i·(base^t)·C
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

func (vrp *ValueRangeProof) Validate(ac *AssetCommitment, vc *ValueCommitment, msg []byte) bool {
	if vrp.exp > 10 {
		return false
	}
	if vrp.vmin >= 1<<63 {
		return false
	}
	if vrp.nbits%1 != 0 {
		return false
	}
	if vrp.nbits+vrp.exp*4 > 64 {
		return false
	}
	p10 := uint64(1)
	for i := uint8(0); i < vrp.exp; i++ {
		p10 *= 10
	}
	if vrp.vmin+p10*((1<<vrp.nbits)-1) >= 1<<63 {
		return false
	}
	n := vrp.nbits / 2
	msghash := vrpMsgHash(ac, vc, vrp.nbits, vrp.exp, vrp.vmin, msg)

	// 5. Calculate last digit commitment `D[n-1] = (10^(-exp))·(VC.V - vmin·AC.H) - ∑(D[t])`, where `∑(D[t])` is a sum of all but the last digit commitment specified in the input to this algorithm.
	var lastDigit ecmath.Point
	var vminScalar ecmath.Scalar
	vminScalar.SetUint64(vrp.vmin)
	lastDigit.ScMul(&ac[0], &vminScalar)               // lastDigit = vmin·AC.H
	lastDigit.Sub(&vc[0], &lastDigit)                  // lastDigit = VC.V - vmin·AC.H
	lastDigit.ScMul(&lastDigit, &powersOf10[-vrp.exp]) // lastDigit = (10^(-exp))·(VC.V - vmin·AC.H)
	dsum := ecmath.ZeroPoint
	for i := 0; i < len(vrp.digits)-1; i++ {
		dsum.Add(&dsum, &vrp.digits[i][0])
	}
	lastDigit.Sub(&lastDigit, &dsum) // lastDigit = (10^(-exp))·(VC.V - vmin·AC.H) - ∑(D[t])
	baseToTheT := uint64(1)
	P := make([][][]ecmath.Point, n)
	for t := uint8(0); t < n; t++ {
		P[t] = make([][]ecmath.Point, base)

		var baseToTheTScalar ecmath.Scalar
		baseToTheTScalar.SetUint64(baseToTheT)

		var baseToTheTH, baseToTheTC ecmath.Point
		baseToTheTH.ScMul(&ac[0], &baseToTheTScalar)
		baseToTheTC.ScMul(&ac[1], &baseToTheTScalar)

		iBaseToTheTH := ecmath.ZeroPoint
		iBaseToTheTC := ecmath.ZeroPoint

		digit := vrp.digits[t]
		if t == n-1 {
			digit[0] = lastDigit
		}

		for i := 0; i < base; i++ {
			P[t][i] = make([]ecmath.Point, 2)
			copy(P[t][i][:], digit[:])
			if i > 0 {
				P[t][i][0].Sub(&P[t][i][0], &iBaseToTheTH)
				P[t][i][1].Sub(&P[t][i][1], &iBaseToTheTC)
			}
			iBaseToTheTH.Add(&iBaseToTheTH, &baseToTheTH)
			iBaseToTheTC.Add(&iBaseToTheTC, &baseToTheTC)
		}

		baseToTheT *= base
	}
	return vrp.brs.Validate(msghash[:], []ecmath.Point{G, J}, P)
}

func (vrp *ValueRangeProof) Payload(ac *AssetCommitment, vc *ValueCommitment, value uint64, f ecmath.Scalar, idek DataKey, vek ValueKey, msg []byte) [][32]byte {
	return nil // xxx
}

func vrpMsgHash(ac *AssetCommitment, vc *ValueCommitment, N uint8, exp uint8, vmin uint64, msg []byte) [32]byte {
	return hash256("ChainCA.VRP", ac.Bytes(), vc.Bytes(), uint64le(uint64(N)), uint64le(uint64(exp)), uint64le(vmin), msg)
}
