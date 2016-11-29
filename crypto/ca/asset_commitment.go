package ca

import "io"

type (
	AssetCommitment Point
	AssetID         [32]byte // not using bc.AssetID to avoid a circular dependency
)

func (ac *AssetCommitment) Bytes() []byte {
	return (*Point)(ac).bytes()
}

func (ac *AssetCommitment) writeTo(w io.Writer) error {
	return (*Point)(ac).WriteTo(w)
}

func (ac *AssetCommitment) readFrom(r io.Reader) error {
	return (*Point)(ac).readFrom(r)
}

func (ac *AssetCommitment) FromBytes(b *[32]byte) error {
	return (*Point)(ac).fromBytes(b)
}

func (ac AssetCommitment) MarshalText() ([]byte, error) {
	return Point(ac).MarshalText()
}

func (ac *AssetCommitment) UnmarshalText(b []byte) error {
	return (*Point)(ac).UnmarshalText(b)
}

// Creates a nonblinded asset commitment out of a cleartext AssetID.
func CreateNonblindedAssetCommitment(assetID AssetID) AssetCommitment {
	var P Point

	for ctr := uint64(0); true; ctr++ {
		// 1. Calculate `SHA3-256(assetID || counter64le)`
		h := hash256(assetID[:], uint64le(ctr))

		// 2. Decode the resulting hash as a point `P` on the elliptic curve.
		err := P.fromBytes(&h)

		if err != nil {
			continue
		}

		// 3. Calculate point `A = 8*P` (8 is a cofactor in edwards25519) which belongs to a subgroup of `G` with order `L`.
		P.mul(&cofactor)

		break
	}
	// 4. Return `A`.
	return AssetCommitment(P)
}

// Creates a blinded asset commitment out of a previous asset commitment,
// cumulative blinding factor and AssetID Encryption Key (AEK).
func CreateBlindedAssetCommitment(
	Hprev AssetCommitment,
	cprev Scalar,
	aek AssetKey,
) (AssetCommitment, Scalar) {
	// Note: blinding factor is based on a combined secret based on the
	// secret known to the sender of the input (`cprev`) and the secret
	// known to the recipient (`aek`). This allows to keep the asset ID
	// obfuscated against each of these parties and use a deterministic
	// stateless secret. If the two parties collude, then they can prove
	// to each other the asset ID on the given input and output even
	// without unwinding the ring signature, so the possibility of
	// collusion does not reduce the security of the scheme.

	// 1. Calculate `secret = SHA3-512(cprev || aek)` where `cprev` is encoded as 256-bit little-endian integer.
	// 2. Calculate differential blinding factor by reducing secret modulo subgroup order `L`: `d = secret mod L`.
	d := computeDifferentialBlindingFactor(cprev, aek)

	// 3. Calculate new asset ID commitment: `H’ = H[j] + d*G` and encode it as [public key](data.md#public-key).
	H := blindAssetCommitment(Hprev, d)

	// 4. Return `(H’,d)`.
	return H, d
}

func computeDifferentialBlindingFactor(cprev Scalar, aek AssetKey) Scalar {
	// 1. Calculate `secret = SHA3-512(cprev || aek)` where `cprev` is encoded as 256-bit little-endian integer.
	secret := hash512(cprev[:], aek[:])

	// 2. Calculate differential blinding factor by reducing secret modulo subgroup order `L`: `d = secret mod L`.
	d := reducedScalar(secret)
	return d
}

func blindAssetCommitment(Hprev AssetCommitment, d Scalar) AssetCommitment {
	// 3. Calculate new asset ID commitment: `H’ = H[j] + d*G` and encode it as [public key](data.md#public-key).
	return AssetCommitment(multiplyAndAddPoint(one, Point(Hprev), d))
}

func (ac *AssetCommitment) String() string {
	return (*Point)(ac).String()
}
