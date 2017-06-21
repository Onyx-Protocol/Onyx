package ca

import (
	"bytes"
	"errors"

	"chain/crypto/ed25519/ecmath"
)

// ValueRangeProof is a confidential value range proof.
type ValueRangeProof struct {
	N, exp, vmin uint64
	digits       []PointPair
	brs          *BorromeanRingSignature
}

const base = 4

// CreateValueRangeProof creates a confidential value range proof.
func CreateValueRangeProof(AC *AssetCommitment, VC *ValueCommitment, N, value uint64, pt [][32]byte, f ecmath.Scalar, idek DataKey, vek ValueKey, msg []byte) *ValueRangeProof {
	if uint64(len(pt)) != 2*N-1 {
		panic("calling error")
	}
	switch N {
	case 8, 16, 32, 48, 64:
		// do nothing
	default:
		panic("calling error")
	}
	if value >= 1<<N {
		panic("calling error")
	}

	msghash := vrpMsgHash(AC, VC, N, 0, 0, msg)
	pek := hash256("ChainCA.VRP.pek", msghash[:], idek, f[:], VC.Bytes())
	n := N / 2

	buf := make([]byte, 32*(2*N-1))
	for i := range pt {
		copy(buf[32*i:32*(i+1)], pt[i][:])
	}
	ep := EncryptPacket(pek[:], nil, buf[:32*(2*N-1)])
	ct := make([][32]byte, 2*N)
	for i := uint64(0); i < 2*N-1; i++ {
		copy(ct[i][:], ep.ct[32*i:32*(i+1)])
	}
	copy(ct[2*N-1][:8], ep.nonce[:])
	copy(ct[2*N-1][8:], ep.mac[:])

	b := vrpCalcb(n, msghash, f)

	DB := make([]PointPair, n) // DB[t][0] is what the spec calls D[t], DB[t][1] is B[t]
	j := make([]uint64, n)

	PQ := vrpCalcPQ(AC, n, func(t uint64) PointPair {
		digitVal := value & (0x03 << (2 * t))
		var d ecmath.Scalar
		d.SetUint64(digitVal)

		DB[t][0].ScMulAdd(AC.H(), &d, &b[t]) // D[t] = digit[t]·H + b[t]·G
		DB[t][1].ScMul(AC.C(), &d)           // B[t] = digit[t]·C
		var T ecmath.Point
		T.ScMul(&J, &b[t])
		DB[t][1].Add(&DB[t][1], &T) // B[t] = digit[t]·C + b[t]·J

		j[t] = digitVal >> (2 * t)

		return DB[t]
	})

	brs := CreateBorromeanRingSignature(msghash[:], []ecmath.Point{G, J}, PQ, b, j, ct)
	return &ValueRangeProof{
		N:      N,
		exp:    0,
		vmin:   0,
		digits: DB[:n-1],
		brs:    brs,
	}
}

func (vrp *ValueRangeProof) Validate(ac *AssetCommitment, vc *ValueCommitment, msg []byte) bool {
	if err := vrp.check(); err != nil {
		return false
	}
	n := vrp.N / 2
	msghash := vrpMsgHash(ac, vc, vrp.N, vrp.exp, vrp.vmin, msg)

	// 5. Calculate last digit commitment (D[n-1],B[n-1]) = (10^(-exp))·(VC - vmin·AC) - ∑(D[t],B[t])
	lastDB := vrp.calcLastDB(ac, vc)

	PQ := vrpCalcPQ(ac, n, func(t uint64) PointPair {
		if t == n-1 {
			return lastDB
		}
		return vrp.digits[t]
	})
	return vrp.brs.Validate(msghash[:], []ecmath.Point{G, J}, PQ)
}

func (vrp *ValueRangeProof) Payload(ac *AssetCommitment, vc *ValueCommitment, value uint64, f ecmath.Scalar, idek DataKey, vek ValueKey, msg []byte) [][32]byte {
	if err := vrp.check(); err != nil {
		// xxx error
	}
	n := vrp.N / 2
	msghash := vrpMsgHash(ac, vc, vrp.N, vrp.exp, vrp.vmin, msg)
	lastDB := vrp.calcLastDB(ac, vc)
	j := make([]uint64, n)
	PQ := vrpCalcPQ(ac, n, func(t uint64) PointPair {
		digitVal := value & (0x03 << (2 * t))
		j[t] = digitVal >> (2 * t)
		if t == n-1 {
			return lastDB
		}
		return vrp.digits[t]
	})
	b := vrpCalcb(n, msghash, f)
	pre := vrp.brs.Payload(msghash[:], []ecmath.Point{G, J}, PQ, b, j)
	pek := hash256("ChainCA.VRP.pek", msghash[:], idek, f[:], vc.Bytes())

	buf := new(bytes.Buffer)
	for i := 0; i < len(pre)-1; i++ {
		buf.Write(pre[i][:])
	}
	ep := EncryptedPacket{
		ct: buf.Bytes(),
	}
	copy(ep.nonce[:], pre[len(pre)-1][:8])
	copy(ep.mac[:], pre[len(pre)-1][8:])
	post, ok := ep.Decrypt(pek[:])
	if !ok {
		// xxx error
	}
	pt := make([][32]byte, 2*vrp.N-1)
	for i := uint64(0); i < 2*vrp.N-1; i++ {
		copy(pt[i][:], post[32*i:32*(i+1)])
	}
	return pt
}

func vrpMsgHash(ac *AssetCommitment, vc *ValueCommitment, N, exp, vmin uint64, msg []byte) [32]byte {
	return hash256("ChainCA.VRP", ac.Bytes(), vc.Bytes(), uint64le(uint64(N)), uint64le(uint64(exp)), uint64le(vmin), msg)
}

func (vrp *ValueRangeProof) calcLastDB(ac *AssetCommitment, vc *ValueCommitment) PointPair {
	var (
		vminScalar ecmath.Scalar
		lastDB     PointPair
	)
	vminScalar.SetUint64(vrp.vmin)
	lastDB.ScMul((*PointPair)(ac), &vminScalar)     // lastDB = vmin·AC
	lastDB.Sub((*PointPair)(vc), &lastDB)           // lastDB = VC - vmin·AC
	lastDB.ScMul(&lastDB, powerOf10(-int(vrp.exp))) // lastDB = (10^(-exp))·(VC - vmin·AC)
	dbSum := ZeroPointPair
	for _, digit := range vrp.digits {
		dbSum.Add(&dbSum, &digit)
	}
	lastDB.Sub(&lastDB, &dbSum) // lastDB = (10^(-exp))·(VC - vmin·AC) - ∑(D[t],B[t])
	return lastDB
}

func vrpCalcPQ(ac *AssetCommitment, n uint64, getDigit func(uint64) PointPair) [][][]ecmath.Point {
	PQ := make([][][]ecmath.Point, n)
	baseToTheT := uint64(1)
	for t := uint64(0); t < n; t++ {
		PQ[t] = make([][]ecmath.Point, base)

		var baseToTheTScalar ecmath.Scalar
		baseToTheTScalar.SetUint64(baseToTheT)

		var baseToTheTH, baseToTheTC ecmath.Point
		baseToTheTH.ScMul(ac.H(), &baseToTheTScalar)
		baseToTheTC.ScMul(ac.C(), &baseToTheTScalar)

		iBaseToTheTH := ecmath.ZeroPoint
		iBaseToTheTC := ecmath.ZeroPoint

		digit := getDigit(t)

		for i := 0; i < base; i++ {
			PQ[t][i] = make([]ecmath.Point, 2)
			copy(PQ[t][i][:], digit[:])
			if i > 0 {
				PQ[t][i][0].Sub(&PQ[t][i][0], &iBaseToTheTH)
				PQ[t][i][1].Sub(&PQ[t][i][1], &iBaseToTheTC)
			}
			if i < base-1 {
				iBaseToTheTH.Add(&iBaseToTheTH, &baseToTheTH)
				iBaseToTheTC.Add(&iBaseToTheTC, &baseToTheTC)
			}
		}
		baseToTheT *= base
	}
	return PQ
}

func vrpCalcb(n uint64, msghash [32]byte, f ecmath.Scalar) []ecmath.Scalar {
	b := make([]ecmath.Scalar, n)
	bsum := ecmath.Zero
	hasher := streamHash("ChainCA.VRP.b", msghash[:], f[:])
	for t := uint64(0); t < n-1; t++ {
		var bt [64]byte
		hasher.Read(bt[:])
		b[t].Reduce(&bt)
		bsum.Add(&bsum, &b[t])
	}
	b[n-1].Sub(&f, &bsum)
	return b
}

var vrpErr = errors.New("value range proof error")

func (vrp *ValueRangeProof) check() error {
	if vrp.exp > 10 {
		return vrpErr
	}
	if vrp.vmin >= 1<<63 {
		return vrpErr
	}
	if vrp.N%1 != 0 {
		return vrpErr
	}
	if vrp.N+vrp.exp*4 > 64 {
		return vrpErr
	}

	p10 := uint64(1)
	for i := uint64(0); i < vrp.exp; i++ {
		p10 *= 10
	}
	if vrp.vmin+p10*((1<<vrp.N)-1) >= 1<<63 {
		return vrpErr
	}

	return nil
}
