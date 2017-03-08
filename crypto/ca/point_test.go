package ca

import "testing"

var B = G

func (p *Point) flipBits(f func()) {
	saved := *p
	defer func() { *p = saved }()

	p.add(&B)
	f()
}

func TestGeneratorG(t *testing.T) {

	want := fromHex("5866666666666666666666666666666666666666666666666666666666666666")
	got := G.bytes()

	if !constTimeEqual(want, got) {
		t.Errorf("G is %x, but must be %x", got, want)
	}
}

func TestGeneratorJ(t *testing.T) {

	want := fromHex("00c774b875ed4e395ebb0782b4d93db838d3c4c0840bc970570517555ca71b77")
	got := J.bytes()

	if !constTimeEqual(want, got) {
		t.Errorf("J is %x, but must be %x", got, want)
	}
}

func TestBasePointArith(t *testing.T) {
	B1 := multiplyPoint(one, B)
	if !B.equal(&B1) {
		t.Errorf("B [%x] != 1*B [%x]", B.bytes(), B1.bytes())
	}

	two := addScalars(one, one)
	BB := B
	BB.add(&B)
	B2a := multiplyBasePoint(two)
	B2b := multiplyPoint(two, B)
	if !BB.equal(&B2a) {
		t.Errorf("B+B [%x] != 2B [%x] (using multiplyBasePoint)", BB.bytes(), B2a.bytes())
	}
	if !BB.equal(&B2b) {
		t.Errorf("B+B [%x] != 2B [%x] (using multiplyPoint)", BB.bytes(), B2b.bytes())
	}
	if !B2a.equal(&B2b) {
		t.Errorf("2B [%x] (using multiplyBasePoint) != 2B [%x] (using multiplyPoint)", B2a.bytes(), B2b.bytes())
	}
}

func TestSimpleSchnorr(t *testing.T) {
	// Sign(p,msg):
	//   k = random
	//   R1 = k*G
	//   e1 = Hash(Encode(R1), msg)
	//   s = k + e1*p
	//
	// Verify(e,s,P=p*G, msg):
	//   R2 = s*G - e*P
	//   e2 == Hash(Encode(R2), msg)

	msg := hash256([]byte("attack at dawn"))
	p := reducedScalar(hash512([]byte("privkey")))
	P := multiplyBasePoint(p)

	// Sign:
	k := reducedScalar(hash512([]byte("random nonce")))
	R1 := multiplyBasePoint(k)
	R1enc := encodePoint(&R1)
	e1 := reducedScalar(hash512(R1.bytes(), msg[:]))
	s := multiplyAndAddScalars(e1, p, k)

	// Verify:
	R2 := multiplyAndAddPoint(negateScalar(e1), P, s)
	R2enc := encodePoint(&R2)
	e2 := reducedScalar(hash512(R2.bytes(), msg[:]))

	if R1enc != R2enc {
		t.Fatalf("Intermediate R1 and R2 are not equal:\n R1=%x\n R2=%x", R1enc, R2enc)
	}

	if e2 != e1 {
		t.Fatalf("Schnorr signature is not valid")
	}
}
