// This is an extension to ed25519.Sign that is compatible with NaCl `crypto_sign`
// function taking 64-byte expanded private key (where the left part is a pre-multiplied
// scalar and the right part is "prefix" used for generating a nonce).
//
// Invariants:
// 1) Expanded(PrivateKey).Sign() == PrivateKey.Sign()
// 2) InnerSign(Expanded(PrivateKey)) == Sign(PrivateKey)
package chainkd

import (
	"crypto"
	"crypto/sha512"
	"errors"
	"io"
	"strconv"
	
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/internal/edwards25519"
)

const (
	// ExpandedPrivateKeySize is the size, in bytes, of a "secret key" as defined in NaCl.
	ExpandedPrivateKeySize = 64
)

// ExpandedPrivateKey is the type of NaCl secret keys. It implements crypto.Signer.
type ExpandedPrivateKey []byte

// Public returns the PublicKey corresponding to secret key.
func (priv ExpandedPrivateKey) Public() crypto.PublicKey {
	var A edwards25519.ExtendedGroupElement
	var scalar [32]byte
	copy(scalar[:], priv[:32])
	edwards25519.GeScalarMultBase(&A, &scalar)
	var publicKeyBytes [32]byte
	A.ToBytes(&publicKeyBytes)
	return ed25519.PublicKey(publicKeyBytes[:])
}

func ExpandEd25519PrivateKey(priv ed25519.PrivateKey) ExpandedPrivateKey {
	digest := sha512.Sum512(priv[:32])
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64
	return ExpandedPrivateKey(digest[:])
}

// Sign signs the given message with expanded private key.
// Ed25519 performs two passes over messages to be signed and therefore cannot
// handle pre-hashed messages. Thus opts.HashFunc() must return zero to
// indicate the message hasn't been hashed. This can be achieved by passing
// crypto.Hash(0) as the value for opts.
func (priv ExpandedPrivateKey) Sign(rand io.Reader, message []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	if opts.HashFunc() != crypto.Hash(0) {
		return nil, errors.New("ed25519: cannot sign hashed message")
	}

	return Ed25519InnerSign(priv, message), nil
}

// InnerSign signs the message with expanded private key and returns a signature.
// It will panic if len(privateKey) is not ExpandedPrivateKeySize.
func Ed25519InnerSign(privateKey ExpandedPrivateKey, message []byte) []byte {
	if l := len(privateKey); l != ExpandedPrivateKeySize {
		panic("ed25519: bad private key length: " + strconv.Itoa(l))
	}

	var messageDigest, hramDigest [64]byte

	h := sha512.New()
	h.Write(privateKey[32:])
	h.Write(message)
	h.Sum(messageDigest[:0])

	var messageDigestReduced [32]byte
	edwards25519.ScReduce(&messageDigestReduced, &messageDigest)
	var R edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&R, &messageDigestReduced)

	var encodedR [32]byte
	R.ToBytes(&encodedR)

	publicKey := privateKey.Public().(ed25519.PublicKey)
	h.Reset()
	h.Write(encodedR[:])
	h.Write(publicKey[:])
	h.Write(message)
	h.Sum(hramDigest[:0])
	var hramDigestReduced [32]byte
	edwards25519.ScReduce(&hramDigestReduced, &hramDigest)

	var sk [32]byte
	copy(sk[:], privateKey[:32])
	var s [32]byte
	edwards25519.ScMulAdd(&s, &hramDigestReduced, &sk, &messageDigestReduced)

	signature := make([]byte, ed25519.SignatureSize)
	copy(signature[:], encodedR[:])
	copy(signature[32:], s[:])

	return signature
}
