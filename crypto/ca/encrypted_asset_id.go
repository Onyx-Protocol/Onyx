package ca

import (
	"errors"
	"fmt"
	"io"
)

type EncryptedAssetID struct {
	AssetID        [32]byte
	BlindingFactor [32]byte
}

func (ea *EncryptedAssetID) readFrom(r io.Reader) error {
	_, err := io.ReadFull(r, ea.AssetID[:])
	if err != nil {
		return err
	}
	_, err = io.ReadFull(r, ea.BlindingFactor[:])
	return err
}

func (ea EncryptedAssetID) String() string {
	return fmt.Sprintf("{AssetID: %x; BlindingFactor: %x}", ea.AssetID[:], ea.BlindingFactor[:])
}

func EncryptAssetID(
	assetID AssetID,
	H AssetCommitment,
	c Scalar,
	aek AssetKey,
) (result EncryptedAssetID) {
	// 1. Expand the encryption key: `ek = SHA3-512(aek || H’)`, split the resulting hash in two halves.
	ek := hash512(aek[:], H.Bytes())

	// 2. Encrypt the asset ID using the first half: `ea = assetID XOR ek[0,32]`.
	result.AssetID = xor256(assetID[:], ek[0:32])

	// 3. Encrypt the cumulative blinding factor using the second half: `ec = c XOR ek[32,32]` where `c` is encoded as a 256-bit little-endian integer.
	result.BlindingFactor = xor256(c[:], ek[32:64])

	// 4. Return `(ea,ec)`
	return result
}

func (enc EncryptedAssetID) Decrypt(
	H AssetCommitment,
	aek AssetKey,
) (assetID AssetID, c Scalar, err error) {

	// 1. Expand the encryption key: `ek = SHA3-512(aek || H’)`, split the resulting hash in two halves.
	ek := hash512(aek[:], H.Bytes())

	// 2. Decrypt the asset ID using the first half: `assetID = ea XOR ek[0,32]`.
	assetID = AssetID(xor256(enc.AssetID[:], ek[0:32]))

	// 3. Decrypt the cumulative blinding factor using the second half: `c = ec XOR ek[32,32]`.
	c = xor256(enc.BlindingFactor[:], ek[32:64])

	// 4. Calculate `A` as a nonblinded asset ID commitment:
	//    an elliptic curve point `8*decode(SHA3-256(assetID))`.
	A := CreateNonblindedAssetCommitment(assetID)

	// 5. Calculate point `P = A + c*G` where `c` is interpreted as a little-endian 256-bit integer.
	P := multiplyAndAddPoint(one, Point(A), c)

	// 6. Verify that `P` equals target commitment `H’`. If not, halt and return `nil`.
	if encodePoint(&P) != encodePoint((*Point)(&H)) {
		return assetID, c, errors.New("Asset ID decryption failed")
	}

	// 7. Return `(assetID, c)`.
	return assetID, c, nil
}

func (e *EncryptedAssetID) WriteTo(w io.Writer) error {
	_, err := w.Write(e.AssetID[:])
	if err != nil {
		return err
	}
	_, err = w.Write(e.BlindingFactor[:])
	return err
}
