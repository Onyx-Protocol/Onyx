package ca

import (
	"io"

	"chain/crypto/ed25519/ecmath"
)

type (
	// AssetID is a 32-byte unique identifier of an asset type.
	// We are not using blockchain type AssetID to avoid a circular dependency.
	AssetID [32]byte

	// AssetPoint is an orthogonal generator to which an AssetID is mapped.
	AssetPoint ecmath.Point

	// AssetCommitment is a point pair representing an ElGamal commitment to an AssetPoint.
	AssetCommitment PointPair
)

// Bytes returns binary representation of an asset commitment (64-byte slice)
func (ac *AssetCommitment) Bytes() []byte {
	return (*PointPair)(ac).Bytes()
}

// String returns hex representation of an asset commitment
func (ac *AssetCommitment) String() string {
	return (*PointPair)(ac).String()
}

// MarshalBinary encodes the receiver into a binary form and returns the result (32-byte slice).
func (ac *AssetCommitment) MarshalBinary() ([]byte, error) {
	return (*PointPair)(ac).MarshalBinary()
}

// UnmarshalBinary decodes an asset commitment for a given slice.
// Returns error if the slice is not 32-byte long or the encoding is invalid.
func (ac *AssetCommitment) UnmarshalBinary(data []byte) error {
	return (*PointPair)(ac).UnmarshalBinary(data)
}

// WriteTo writes 32-byte encoding of an asset commitment.
func (ac *AssetCommitment) WriteTo(w io.Writer) (n int64, err error) {
	return (*PointPair)(ac).WriteTo(w)
}

// ReadFrom attempts to read 32 bytes and decode an asset commitment.
func (ac *AssetCommitment) ReadFrom(r io.Reader) (n int64, err error) {
	return (*PointPair)(ac).ReadFrom(r)
}

// MarshalText returns a hex-encoded asset commitment.
func (ac *AssetCommitment) MarshalText() ([]byte, error) {
	return (*PointPair)(ac).MarshalText()
}

// UnmarshalText decodes an asset commitment from a hex-encoded buffer.
func (ac *AssetCommitment) UnmarshalText(data []byte) error {
	return (*PointPair)(ac).UnmarshalText(data)
}

// CreateAssetPoint converts an asset ID to a valid orthogonal generator on EC.
func CreateAssetPoint(assetID AssetID) AssetPoint {
	var P ecmath.Point

	for ctr := uint64(0); true; ctr++ {
		// 1. Calculate `Hash256("AssetID" || assetID || uint64le(counter))`
		h := hash256([]byte("AssetID"), assetID[:], uint64le(ctr))

		// 2. Decode the resulting hash as a point `P` on the elliptic curve.
		err := P.UnmarshalBinary(h[:])

		if err != nil {
			continue
		}

		// 3. Calculate point `A = 8*P` (8 is a cofactor in edwards25519) which belongs to a subgroup of `G` with order `L`.
		cofactor := ecmath.Scalar{8}
		P.ScMul(&P, &cofactor)

		break
	}
	// 4. Return `A`.
	return AssetPoint(P)
}

// CreateNonblindedAssetCommitment creates a nonblinded asset commitment out of an asset point.
func CreateNonblindedAssetCommitment(ap AssetPoint) AssetCommitment {
	pp := ZeroPointPair
	pp.Point1 = ecmath.Point(ap)
	return AssetCommitment(pp)
}

// // Creates a blinded asset commitment out of a previous asset commitment,
// // cumulative blinding factor and AssetID Encryption Key (AEK).
// func CreateBlindedAssetCommitment(
// 	Hprev AssetCommitment,
// 	cprev Scalar,
// 	aek AssetKey,
// ) (AssetCommitment, Scalar) {
// 	// Note: blinding factor is based on a combined secret based on the
// 	// secret known to the sender of the input (`cprev`) and the secret
// 	// known to the recipient (`aek`). This allows to keep the asset ID
// 	// obfuscated against each of these parties and use a deterministic
// 	// stateless secret. If the two parties collude, then they can prove
// 	// to each other the asset ID on the given input and output even
// 	// without unwinding the ring signature, so the possibility of
// 	// collusion does not reduce the security of the scheme.

// 	// 1. Calculate `secret = SHA3-512(cprev || aek)` where `cprev` is encoded as 256-bit little-endian integer.
// 	// 2. Calculate differential blinding factor by reducing secret modulo subgroup order `L`: `d = secret mod L`.
// 	d := computeDifferentialBlindingFactor(cprev, aek)

// 	// 3. Calculate new asset ID commitment: `H’ = H[j] + d*G` and encode it as [public key](data.md#public-key).
// 	H := blindAssetCommitment(Hprev, d)

// 	// 4. Return `(H’,d)`.
// 	return H, d
// }

// func computeDifferentialBlindingFactor(cprev Scalar, aek AssetKey) Scalar {
// 	// 1. Calculate `secret = SHA3-512(cprev || aek)` where `cprev` is encoded as 256-bit little-endian integer.
// 	secret := hash512(cprev[:], aek[:])

// 	// 2. Calculate differential blinding factor by reducing secret modulo subgroup order `L`: `d = secret mod L`.
// 	d := reducedScalar(secret)
// 	return d
// }

// func blindAssetCommitment(Hprev AssetCommitment, d Scalar) AssetCommitment {
// 	// 3. Calculate new asset ID commitment: `H’ = H[j] + d*G` and encode it as [public key](data.md#public-key).
// 	return AssetCommitment(multiplyAndAddPoint(one, Point(Hprev), d))
// }
