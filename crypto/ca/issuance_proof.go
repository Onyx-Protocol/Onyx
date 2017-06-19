package ca

import "chain/crypto/ed25519/ecmath"

type IssuanceProof struct {
	X, Z, Zprime   ecmath.Point
	e1, s1, e2, s2 ecmath.Scalar
}

func CreateIssuanceProof(ac *AssetCommitment, iarp *ConfidentialIARP, a []AssetID, msg []byte, nonce [32]byte, y ecmath.Scalar, Y ecmath.Point) *IssuanceProof {
	// 1. [Validate issuance asset range proof](#validate-issuance-asset-range-proof) to make sure tracing and marker points are correct.
	if !iarp.Validate(ac, a, iarp.Y, nonce, msg) {
		return nil // xxx or panic?
	}

	// 2. Calculate the blinding scalar x = ScalarHash("x", AC, T, y, nonce, message)
	x := scalarHash("ChainCA.x", ac.Bytes(), iarp.T.Bytes(), y[:], nonce[:], msg)

	// 3. Blind the tracing point being tested: `Z = x·T`.
	var Z ecmath.Point
	Z.ScMul(&iarp.T, &x)

	M := iarpCalcM(iarpBasehash(ac, nonce, msg, a, iarp.Y))

	// 4. Calculate commitment to the blinding key: `X = x·M`.
	var X ecmath.Point
	X.ScMul(&M, &x)

	// 5. Calculate and blind a tracing point corresponding to the issuance key pair `y,Y`: `Z’ = x·y·M`.
	var (
		Zprime ecmath.Point
		xy     ecmath.Scalar
	)
	xy.Mul(&x, &y)
	Zprime.ScMul(&M, &xy)

	// 6. Calculate a message hash: `msghash = Hash256("IP", AC, T, X, Z, Z’)`.
	msghash := hash256("ChainCA.IP", ac.Bytes(), iarp.T.Bytes(), X.Bytes(), Z.Bytes(), Zprime.Bytes())

	// 7. Create a proof that `Z` blinds tracing point `T` and `X` commits to that blinding factor (i.e. the discrete log `X/M` is equal to the discrete log `Z/T`):
	// 7.1. Calculate the nonce `k1 = ScalarHash("k1", msghash, y, x)`
	k1 := scalarHash("ChainCA.k1", msghash[:], y[:], x[:])
	// 7.2. Calculate point `R1 = k1·M`.
	var R1, R2 ecmath.Point
	R1.ScMul(&M, &k1)
	// 7.3. Calculate point `R2 = k1·T`.
	R2.ScMul(&iarp.T, &k1)
	// 7.4. Calculate scalar `e1 = ScalarHash("e1", msghash, R1, R2)`.
	e1 := scalarHash("ChainCA.e1", msghash[:], R1.Bytes(), R2.Bytes())
	var s1 ecmath.Scalar
	s1.MulAdd(&x, &e1, &k1)

	// 8. Create a proof that `Z’` is a blinded tracing point corresponding to `Y[j]` (i.e. the discrete log `Z’/X` is equal to the discrete log `Y[j]/G`):
	// 8.1. Calculate the nonce `k2 = ScalarHash("k2", msghash, y, x)`.
	k2 := scalarHash("ChainCA.k2", msghash[:], y[:], x[:])
	// 8.2. Calculate point `R3 = k2·X`.
	var R3, R4 ecmath.Point
	R3.ScMul(&X, &k2)
	// 8.3. Calculate point `R4 = k2·G`.
	R4.ScMul(&G, &k2)
	// 8.4. Calculate scalar `e2 = ScalarHash("e2", msghash, R3, R4)`.
	e2 := scalarHash("ChainCA.e2", msghash[:], R3.Bytes(), R4.Bytes())
	var s2 ecmath.Scalar
	s2.MulAdd(&y, &e2, &k2)
	return &IssuanceProof{X: X, Z: Z, Zprime: Zprime, e1: e1, s1: s1, e2: e2, s2: s2}
}

// Validate validates ip. It returns two bools: overall validity, and whether Y[j] was used to issue the asset ID in commitment ac.
func (ip *IssuanceProof) Validate(ac *AssetCommitment, iarp *ConfidentialIARP, a []AssetID, msg []byte, nonce [32]byte, j int) (valid, yj bool) {
	if !iarp.Validate(ac, a, iarp.Y, nonce, msg) {
		return false, false
	}
	msghash := hash256("ChainCA.IP", ac.Bytes(), iarp.T.Bytes(), ip.X.Bytes(), ip.Z.Bytes(), ip.Zprime.Bytes())

	M := iarpCalcM(iarpBasehash(ac, nonce, msg, a, iarp.Y))

	var R1, R2, Temp ecmath.Point
	R1.ScMul(&M, &ip.s1)
	Temp.ScMul(&ip.X, &ip.e1)
	R1.Sub(&R1, &Temp) // R1 = s1·M - e1·X
	R2.ScMul(&iarp.T, &ip.s1)
	Temp.ScMul(&ip.Z, &ip.e1)
	R2.Sub(&R2, &Temp) // R2 = s1·T - e1·Z
	ePrime := scalarHash("ChainCA.e1", msghash[:], R1.Bytes(), R2.Bytes())
	if ePrime != ip.e1 {
		return false, false
	}

	var R3, R4 ecmath.Point
	R3.ScMul(&ip.X, &ip.s2)
	Temp.ScMul(&ip.Zprime, &ip.e2)
	R3.Sub(&R3, &Temp) // R3 = s2·X - e2·Z’
	R4.ScMul(&G, &ip.s2)
	Temp.ScMul(&iarp.Y[j], &ip.e2)
	R4.Sub(&R4, &Temp) // R4 = s2·G - e2·Y[j]
	ePrime = scalarHash("ChainCA.e2", msghash[:], R3.Bytes(), R4.Bytes())
	if ePrime != ip.e2 {
		return false, false
	}
	var Z, Zprime ecmath.Point
	Z.ScMul(&ip.Z, &ecmath.Cofactor)
	Zprime.ScMul(&ip.Zprime, &ecmath.Cofactor)
	return true, Z.ConstTimeEqual(&Zprime)
}
