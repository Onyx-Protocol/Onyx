package ca

import (
	"encoding/hex"
	"fmt"
	"io"

	"chain-stealth/crypto/ed25519/edwards25519"
)

type Point edwards25519.ExtendedGroupElement

var ZeroPoint Point
var G = makeG()
var J = makeJ()
var Gi = []Point{
	makeGiPure(0), makeGiPure(1), makeGiPure(2), makeGiPure(3),
	makeGiPure(4), makeGiPure(5), makeGiPure(6), makeGiPure(7),
	makeGiPure(8), makeGiPure(9), makeGiPure(10), makeGiPure(11),
	makeGiPure(12), makeGiPure(13), makeGiPure(14), makeGiPure(15),
	makeGiPure(16), makeGiPure(17), makeGiPure(18), makeGiPure(19),
	makeGiPure(20), makeGiPure(21), makeGiPure(22), makeGiPure(23),
	makeGiPure(24), makeGiPure(25), makeGiPure(26), makeGiPure(27),
	makeGiPure(28), makeGiPure(29), makeGiPure(30),
}

// Precomputed reminder generators: GR[i] = G - Sum[G[j], j<i]; GR[0] = G
var GRi = []Point{
	makeGiPure(0), makeGiPure(1), makeGiPure(2), makeGiPure(3),
	makeGiPure(4), makeGiPure(5), makeGiPure(6), makeGiPure(7),
	makeGiPure(8), makeGiPure(9), makeGiPure(10), makeGiPure(11),
	makeGiPure(12), makeGiPure(13), makeGiPure(14), makeGiPure(15),
	makeGiPure(16), makeGiPure(17), makeGiPure(18), makeGiPure(19),
	makeGiPure(20), makeGiPure(21), makeGiPure(22), makeGiPure(23),
	makeGiPure(24), makeGiPure(25), makeGiPure(26), makeGiPure(27),
	makeGiPure(28), makeGiPure(29), makeGiPure(30),
}

func (a *Point) add(b *Point) {
	var b2 edwards25519.CachedGroupElement
	(*edwards25519.ExtendedGroupElement)(b).ToCached(&b2)

	var c edwards25519.CompletedGroupElement
	edwards25519.GeAdd(&c, (*edwards25519.ExtendedGroupElement)(a), &b2)

	c.ToExtended((*edwards25519.ExtendedGroupElement)(a))
}

func (a *Point) sub(b *Point) {
	var b2 edwards25519.CachedGroupElement
	(*edwards25519.ExtendedGroupElement)(b).ToCached(&b2)

	var c edwards25519.CompletedGroupElement
	edwards25519.GeSub(&c, (*edwards25519.ExtendedGroupElement)(a), &b2)

	c.ToExtended((*edwards25519.ExtendedGroupElement)(a))
}

func (A *Point) mul(x *Scalar) *Point {
	var Rproj edwards25519.ProjectiveGroupElement

	// FIXME: replace with constant-time implementation to avoid sidechannel attacks
	edwards25519.GeDoubleScalarMultVartime(&Rproj, (*[32]byte)(x), (*edwards25519.ExtendedGroupElement)(A), (*[32]byte)(&ZeroScalar))

	var buf [32]byte
	Rproj.ToBytes(&buf)
	(*edwards25519.ExtendedGroupElement)(A).FromBytes(&buf) // xxx check return value? shouldn't be necessary...

	return A
}

func (a *Point) equal(b *Point) bool {
	abuf := encodePoint(a)
	bbuf := encodePoint(b)
	return constTimeEqual(abuf[:], bbuf[:])
}

func (p *Point) bytes() []byte {
	buf := encodePoint(p)
	return buf[:]
}

func (p *Point) fromBytes(inp *[32]byte) error {
	if !(*edwards25519.ExtendedGroupElement)(p).FromBytes(inp) {
		return fmt.Errorf("could not decode point")
	}
	return nil
}

func subPoints(a Point, b Point) Point {
	acopy := a
	acopy.sub(&b)
	return acopy
}

func decodePoint(input [32]byte) (result Point, ok bool) {
	err := result.fromBytes(&input)
	return result, err == nil
}

func encodePoint(pointref *Point) (buf [32]byte) {
	ge := (*edwards25519.ExtendedGroupElement)(pointref)
	ge.ToBytes(&buf)
	return buf
}

// Computes x*P, where P is an arbitrary point on a curve
func multiplyPoint(x Scalar, P Point) Point {
	return multiplyAndAddPoint(x, P, ZeroScalar)
}

// Computes x*G, where G is a base point.
func multiplyBasePoint(x Scalar) (result Point) {
	edwards25519.GeScalarMultBase((*edwards25519.ExtendedGroupElement)(&result), (*[32]byte)(&x))
	return result
}

// Computes a*A + b*G, where G is a base point.
func multiplyAndAddPoint(a Scalar, A Point, b Scalar) Point {
	var Rproj edwards25519.ProjectiveGroupElement
	var Rext edwards25519.ExtendedGroupElement

	// FIXME: replace with constant-time implementation to avoid sidechannel attacks
	edwards25519.GeDoubleScalarMultVartime(&Rproj, (*[32]byte)(&a), (*edwards25519.ExtendedGroupElement)(&A), (*[32]byte)(&b))

	var buf [32]byte
	Rproj.ToBytes(&buf)
	Rext.FromBytes(&buf)
	return Point(Rext)
}

func (p *Point) String() string {
	enc := encodePoint(p)
	return hex.EncodeToString(enc[:])
}

func (p *Point) readFrom(r io.Reader) error {
	var b [32]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return err
	}
	return p.fromBytes(&b)
}

func (p *Point) WriteTo(w io.Writer) error {
	buf := encodePoint(p)
	_, err := w.Write(buf[:])
	return err
}

func (p Point) MarshalText() ([]byte, error) {
	buf := encodePoint(&p)
	res := make([]byte, hex.EncodedLen(len(buf)))
	hex.Encode(res, buf[:])
	return res, nil
}

func (p *Point) UnmarshalText(b []byte) error {
	var buf [32]byte
	if len(b) != hex.EncodedLen(len(buf)) {
		return fmt.Errorf("Point.UnmarshalJSON got input with wrong length %d", len(b))
	}
	_, err := hex.Decode(buf[:], b)
	if err != nil {
		return err
	}
	var ok bool
	*p, ok = decodePoint(buf)
	if !ok {
		return fmt.Errorf("Point.UnmarshalText could not decode point")
	}
	return nil
}

func makeG() Point {
	return multiplyBasePoint(one)
}

func makeJ() (j Point) {
	// Decode the point from SHA3(G)
	h := hash256(G.bytes())
	err := j.fromBytes(&h)
	if err != nil {
		panic("failed to decode secondary generator")
	}
	// Calculate point `J = 8*J` (8 is a cofactor in edwards25519) which belongs to a subgroup of `G` with order `L`.
	j.mul(&cofactor)
	return
}

func makeGiPure(i byte) Point {
	p, _ := makeGi(i)
	return p
}

func makeGi(i byte) (P Point, ctr uint64) {
	Gbytes := G.bytes()
	for ctr = uint64(0); true; ctr++ {
		// 1. Calculate `SHA3-256(i || Encode(G) || counter64le)`
		h := hash256([]byte{i}, Gbytes, uint64le(ctr))

		// 2. Decode the resulting hash as a point `P` on the elliptic curve.
		err := P.fromBytes(&h)

		if err != nil {
			continue
		}

		// 3. Calculate point `G[i] = 8*P` (8 is a cofactor in edwards25519) which belongs to a subgroup of `G` with order `L`.
		P.mul(&cofactor)

		break
	}
	return
}

func init() {
	(*edwards25519.ExtendedGroupElement)(&ZeroPoint).Zero()
}
