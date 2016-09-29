package chainkd

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"hash"
	"io"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/internal/edwards25519"
)

type (
	// TODO(bobg): consider making these types opaque. See https://github.com/chain/chain/pull/1875#discussion_r80577736
	XPrv [64]byte
	XPub [64]byte
)

var one = [32]byte{1}

// NewXPrv takes a source of random bytes and produces a new XPrv. If
// r is nil, crypto/rand.Reader is used.
func NewXPrv(r io.Reader) (xprv XPrv, err error) {
	if r == nil {
		r = rand.Reader
	}
	var entropy [32]byte
	_, err = io.ReadFull(r, entropy[:])
	if err != nil {
		return xprv, err
	}
	hasher := sha512.New()
	hasher.Write([]byte("Chain seed"))
	hasher.Write(entropy[:])
	hasher.Sum(xprv[:0])
	modifyScalar(xprv[:32])
	return xprv, nil
}

func (xprv XPrv) XPub() XPub {
	var buf [32]byte
	copy(buf[:], xprv[:32])

	var P edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&P, &buf)
	P.ToBytes(&buf)

	var xpub XPub
	copy(xpub[:32], buf[:])
	copy(xpub[32:], xprv[32:])

	return xpub
}

func (xprv XPrv) Child(sel []byte, hardened bool) (res XPrv) {
	if hardened {
		hashKeySaltSelector(res[:], 0, xprv[:32], xprv[32:], sel)
		return res
	}

	var s [32]byte
	copy(s[:], xprv[:32])
	var P edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&P, &s)

	var pubkey [32]byte
	P.ToBytes(&pubkey)

	hashKeySaltSelector(res[:], 1, pubkey[:], xprv[32:], sel)

	var (
		f  [32]byte
		s2 [32]byte
	)
	copy(f[:], res[:32])
	edwards25519.ScMulAdd(&s2, &one, &f, &s)
	copy(res[:32], s2[:])

	return res
}

func (xpub XPub) Child(sel []byte) (res XPub) {
	hashKeySaltSelector(res[:], 1, xpub[:32], xpub[32:], sel)

	var (
		f [32]byte
		F edwards25519.ExtendedGroupElement
	)
	copy(f[:], res[:32])
	edwards25519.GeScalarMultBase(&F, &f)

	var (
		pubkey [32]byte
		P      edwards25519.ExtendedGroupElement
	)
	copy(pubkey[:], xpub[:32])
	P.FromBytes(&pubkey)

	var (
		P2 edwards25519.ExtendedGroupElement
		R  edwards25519.CompletedGroupElement
		Fc edwards25519.CachedGroupElement
	)
	F.ToCached(&Fc)
	edwards25519.GeAdd(&R, &P, &Fc)
	R.ToExtended(&P2)

	P2.ToBytes(&pubkey)

	copy(res[:32], pubkey[:])

	return res
}

func (xprv XPrv) Derive(path []uint32) XPrv {
	res := xprv
	for _, p := range path {
		hardened := p >= 1<<31
		var sel [4]byte
		binary.LittleEndian.PutUint32(sel[:], p)
		res = res.Child(sel[:], hardened)
	}
	return res
}

func (xpub XPub) Derive(path []uint32) XPub {
	res := xpub
	for _, p := range path {
		var sel [4]byte
		binary.LittleEndian.PutUint32(sel[:], p)
		res = res.Child(sel[:])
	}
	return res
}

func (xprv XPrv) Sign(msg []byte) []byte {
	var s [32]byte
	copy(s[:], xprv[:32])

	var h [64]byte
	hashKeySalt(h[:], 2, xprv[:32], xprv[32:])

	var P edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&P, &s)

	var pubkey [32]byte
	P.ToBytes(&pubkey)

	var r [64]byte
	hasher := sha512.New()
	hasher.Write(h[:32])
	hasher.Write(msg)
	hasher.Sum(r[:0])

	var rReduced [32]byte
	edwards25519.ScReduce(&rReduced, &r)

	var rPoint edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&rPoint, &rReduced)

	var R [32]byte
	rPoint.ToBytes(&R)

	hasher.Reset()
	hasher.Write(R[:])
	hasher.Write(pubkey[:])
	hasher.Write(msg)

	var k [64]byte
	hasher.Sum(k[:0])

	var kReduced [32]byte
	edwards25519.ScReduce(&kReduced, &k)

	var S [32]byte
	edwards25519.ScMulAdd(&S, &kReduced, &s, &rReduced)

	return append(R[:], S[:]...)
}

func (xpub XPub) Verify(msg []byte, sig []byte) bool {
	return ed25519.Verify(xpub.PublicKey(), msg, sig)
}

// PublicKey extracts the ed25519 public key from an xpub.
func (xpub XPub) PublicKey() ed25519.PublicKey {
	return ed25519.PublicKey(xpub[:32])
}

func hashKeySaltSelector(out []byte, version byte, key, salt, sel []byte) {
	hasher := hashKeySaltHelper(version, key, salt)
	var l [10]byte
	n := binary.PutUvarint(l[:], uint64(len(sel)))
	hasher.Write(l[:n])
	hasher.Write(sel)
	hasher.Sum(out[:0])
	modifyScalar(out)
}

func hashKeySalt(out []byte, version byte, key, salt []byte) {
	hasher := hashKeySaltHelper(version, key, salt)
	hasher.Sum(out[:0])
}

func hashKeySaltHelper(version byte, key, salt []byte) hash.Hash {
	hasher := sha512.New()
	hasher.Write([]byte{version})
	hasher.Write(key)
	hasher.Write(salt)
	return hasher
}

// s must be >= 32 bytes long and gets rewritten in place
func modifyScalar(s []byte) {
	s[0] &= 248
	s[31] &= 127
	s[31] |= 64
}
