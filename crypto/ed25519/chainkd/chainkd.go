package chainkd

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"io"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/ecmath"
)

const (
	// XPubSize is the size, in bytes, of extended public keys.
	XPubSize = 64

	// XPrvSize is the size, in bytes, of extended public keys.
	XPrvSize = 64
)

// XPrv is an opaque type representing an extended private key
type XPrv struct{ data [64]byte }

// XPub is an opaque type representing an extended public key
type XPub struct{ data [64]byte }

// NewXPrv takes a source of random bytes and produces a new XPrv.
// If r is nil, crypto/rand.Reader is used.
func NewXPrv(r io.Reader) (xprv XPrv, err error) {
	if r == nil {
		r = rand.Reader
	}
	var entropy [32]byte
	_, err = io.ReadFull(r, entropy[:])
	if err != nil {
		return xprv, err
	}
	return RootXPrv(entropy[:]), nil
}

// RootXPrv takes a seed binary string and produces a new xprv.
func RootXPrv(seed []byte) (xprv XPrv) {
	h := hmac.New(sha512.New, []byte{'R', 'o', 'o', 't'})
	h.Write(seed)
	h.Sum(xprv.data[:0])
	pruneRootScalar(xprv.data[:32])
	return
}

// XPub derives an extended public key from a given xprv.
func (xprv XPrv) XPub() (xpub XPub) {
	var scalar ecmath.Scalar
	copy(scalar[:], xprv.data[:32])

	var P ecmath.Point
	P.ScMulBase(&scalar)
	buf := P.Encode()

	copy(xpub.data[:32], buf[:])
	copy(xpub.data[32:], xprv.data[32:])

	return
}

// Child derives a child xprv based on `selector` string and `hardened` flag.
// If `hardened` is false, child xpub can be derived independently
// from the parent xpub without using the parent xprv.
// If `hardened` is true, child key can only be derived from the parent xprv.
func (xprv XPrv) Child(sel []byte, hardened bool) XPrv {
	if hardened {
		return xprv.hardenedChild(sel)
	} else {
		return xprv.nonhardenedChild(sel)
	}
}

func (xprv XPrv) hardenedChild(sel []byte) (res XPrv) {
	h := hmac.New(sha512.New, xprv.data[32:])
	h.Write([]byte{'H'})
	h.Write(xprv.data[:32])
	h.Write(sel)
	h.Sum(res.data[:0])
	pruneRootScalar(res.data[:32])
	return
}

func (xprv XPrv) nonhardenedChild(sel []byte) (res XPrv) {
	xpub := xprv.XPub()

	h := hmac.New(sha512.New, xpub.data[32:])
	h.Write([]byte{'N'})
	h.Write(xpub.data[:32])
	h.Write(sel)
	h.Sum(res.data[:0])

	pruneIntermediateScalar(res.data[:32])

	// Unrolled the following loop:
	// var carry int
	// carry = 0
	// for i := 0; i < 32; i++ {
	//         sum := int(xprv.data[i]) + int(res.data[i]) + carry
	//         res.data[i] = byte(sum & 0xff)
	//         carry = (sum >> 8)
	// }

	sum := int(0)

	sum = int(xprv.data[0]) + int(res.data[0]) + (sum >> 8)
	res.data[0] = byte(sum & 0xff)
	sum = int(xprv.data[1]) + int(res.data[1]) + (sum >> 8)
	res.data[1] = byte(sum & 0xff)
	sum = int(xprv.data[2]) + int(res.data[2]) + (sum >> 8)
	res.data[2] = byte(sum & 0xff)
	sum = int(xprv.data[3]) + int(res.data[3]) + (sum >> 8)
	res.data[3] = byte(sum & 0xff)
	sum = int(xprv.data[4]) + int(res.data[4]) + (sum >> 8)
	res.data[4] = byte(sum & 0xff)
	sum = int(xprv.data[5]) + int(res.data[5]) + (sum >> 8)
	res.data[5] = byte(sum & 0xff)
	sum = int(xprv.data[6]) + int(res.data[6]) + (sum >> 8)
	res.data[6] = byte(sum & 0xff)
	sum = int(xprv.data[7]) + int(res.data[7]) + (sum >> 8)
	res.data[7] = byte(sum & 0xff)
	sum = int(xprv.data[8]) + int(res.data[8]) + (sum >> 8)
	res.data[8] = byte(sum & 0xff)
	sum = int(xprv.data[9]) + int(res.data[9]) + (sum >> 8)
	res.data[9] = byte(sum & 0xff)
	sum = int(xprv.data[10]) + int(res.data[10]) + (sum >> 8)
	res.data[10] = byte(sum & 0xff)
	sum = int(xprv.data[11]) + int(res.data[11]) + (sum >> 8)
	res.data[11] = byte(sum & 0xff)
	sum = int(xprv.data[12]) + int(res.data[12]) + (sum >> 8)
	res.data[12] = byte(sum & 0xff)
	sum = int(xprv.data[13]) + int(res.data[13]) + (sum >> 8)
	res.data[13] = byte(sum & 0xff)
	sum = int(xprv.data[14]) + int(res.data[14]) + (sum >> 8)
	res.data[14] = byte(sum & 0xff)
	sum = int(xprv.data[15]) + int(res.data[15]) + (sum >> 8)
	res.data[15] = byte(sum & 0xff)
	sum = int(xprv.data[16]) + int(res.data[16]) + (sum >> 8)
	res.data[16] = byte(sum & 0xff)
	sum = int(xprv.data[17]) + int(res.data[17]) + (sum >> 8)
	res.data[17] = byte(sum & 0xff)
	sum = int(xprv.data[18]) + int(res.data[18]) + (sum >> 8)
	res.data[18] = byte(sum & 0xff)
	sum = int(xprv.data[19]) + int(res.data[19]) + (sum >> 8)
	res.data[19] = byte(sum & 0xff)
	sum = int(xprv.data[20]) + int(res.data[20]) + (sum >> 8)
	res.data[20] = byte(sum & 0xff)
	sum = int(xprv.data[21]) + int(res.data[21]) + (sum >> 8)
	res.data[21] = byte(sum & 0xff)
	sum = int(xprv.data[22]) + int(res.data[22]) + (sum >> 8)
	res.data[22] = byte(sum & 0xff)
	sum = int(xprv.data[23]) + int(res.data[23]) + (sum >> 8)
	res.data[23] = byte(sum & 0xff)
	sum = int(xprv.data[24]) + int(res.data[24]) + (sum >> 8)
	res.data[24] = byte(sum & 0xff)
	sum = int(xprv.data[25]) + int(res.data[25]) + (sum >> 8)
	res.data[25] = byte(sum & 0xff)
	sum = int(xprv.data[26]) + int(res.data[26]) + (sum >> 8)
	res.data[26] = byte(sum & 0xff)
	sum = int(xprv.data[27]) + int(res.data[27]) + (sum >> 8)
	res.data[27] = byte(sum & 0xff)
	sum = int(xprv.data[28]) + int(res.data[28]) + (sum >> 8)
	res.data[28] = byte(sum & 0xff)
	sum = int(xprv.data[29]) + int(res.data[29]) + (sum >> 8)
	res.data[29] = byte(sum & 0xff)
	sum = int(xprv.data[30]) + int(res.data[30]) + (sum >> 8)
	res.data[30] = byte(sum & 0xff)
	sum = int(xprv.data[31]) + int(res.data[31]) + (sum >> 8)
	res.data[31] = byte(sum & 0xff)

	if (sum >> 8) != 0 {
		panic("sum does not fit in 256-bit int")
	}
	return
}

// Child derives a child xpub based on `selector` string.
// The corresponding child xprv can be derived from the parent xprv
// using non-hardened derivation: `parentxprv.Child(sel, false)`.
func (xpub XPub) Child(sel []byte) (res XPub) {
	h := hmac.New(sha512.New, xpub.data[32:])
	h.Write([]byte{'N'})
	h.Write(xpub.data[:32])
	h.Write(sel)
	h.Sum(res.data[:0])

	pruneIntermediateScalar(res.data[:32])

	var (
		f ecmath.Scalar
		F ecmath.Point
	)
	copy(f[:], res.data[:32])
	F.ScMulBase(&f)

	var (
		pubkey [32]byte
		P      ecmath.Point
	)
	copy(pubkey[:], xpub.data[:32])
	_, ok := P.Decode(pubkey)
	if !ok {
		panic("XPub should have been validated on initialization")
	}

	P.Add(&P, &F)
	pubkey = P.Encode()
	copy(res.data[:32], pubkey[:])

	return
}

// Derive generates a child xprv by recursively deriving
// non-hardened child xprvs over the list of selectors:
// `Derive([a,b,c,...]) == Child(a).Child(b).Child(c)...`
func (xprv XPrv) Derive(path [][]byte) XPrv {
	res := xprv
	for _, p := range path {
		res = res.Child(p, false)
	}
	return res
}

// Derive generates a child xpub by recursively deriving
// non-hardened child xpubs over the list of selectors:
// `Derive([a,b,c,...]) == Child(a).Child(b).Child(c)...`
func (xpub XPub) Derive(path [][]byte) XPub {
	res := xpub
	for _, p := range path {
		res = res.Child(p)
	}
	return res
}

// Sign creates an EdDSA signature using expanded private key
// derived from the xprv.
func (xprv XPrv) Sign(msg []byte) []byte {
	return Ed25519InnerSign(xprv.ExpandedPrivateKey(), msg)
}

// Verify checks an EdDSA signature using public key
// extracted from the first 32 bytes of the xpub.
func (xpub XPub) Verify(msg []byte, sig []byte) bool {
	return ed25519.Verify(xpub.PublicKey(), msg, sig)
}

// ExpandedPrivateKey generates a 64-byte key where
// the first half is the scalar copied from xprv,
// and the second half is the `prefix` is generated via PRF
// from the xprv.
func (xprv XPrv) ExpandedPrivateKey() ExpandedPrivateKey {
	var res [64]byte
	h := hmac.New(sha512.New, []byte{'E', 'x', 'p', 'a', 'n', 'd'})
	h.Write(xprv.data[:])
	h.Sum(res[:0])
	copy(res[:32], xprv.data[:32])
	return res[:]
}

// PublicKey extracts the ed25519 public key from an xpub.
func (xpub XPub) PublicKey() ed25519.PublicKey {
	return ed25519.PublicKey(xpub.data[:32])
}

// s must be >= 32 bytes long and gets rewritten in place.
// This is NOT the same pruning as in Ed25519: it additionally clears the third
// highest bit to ensure subkeys do not overflow the second highest bit.
func pruneRootScalar(s []byte) {
	s[0] &= 248
	s[31] &= 31 // clear top 3 bits
	s[31] |= 64 // set second highest bit
}

// Clears lowest 3 bits and highest 23 bits of `f`.
func pruneIntermediateScalar(f []byte) {
	f[0] &= 248 // clear bottom 3 bits
	f[29] &= 1  // clear 7 high bits
	f[30] = 0   // clear 8 bits
	f[31] = 0   // clear 8 bits
}
