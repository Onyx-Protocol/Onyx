package ca

import "testing"

var B = G

func (p *Point) flipBits(f func()) {
	saved := *p
	defer func() { *p = saved }()

	p.add(&B)
	f()
}

func TestZeroPoint(t *testing.T) {

	want := fromHex("0100000000000000000000000000000000000000000000000000000000000000")
	got := ZeroPoint.bytes()

	if !constTimeEqual(want, got) {
		t.Errorf("ZeroPoint is encoded as %x, but must be %x", got, want)
	}
}

func TestGeneratorG(t *testing.T) {

	want := fromHex("5866666666666666666666666666666666666666666666666666666666666666")
	got := G.bytes()

	if !constTimeEqual(want, got) {
		t.Errorf("G is encoded as %x, but must be %x", got, want)
	}
}

func TestGeneratorJ(t *testing.T) {
	want := fromHex("00c774b875ed4e395ebb0782b4d93db838d3c4c0840bc970570517555ca71b77")
	got := J.bytes()

	if !constTimeEqual(want, got) {
		t.Errorf("J is encoded as %x, but must be %x", got, want)
	}
}

func TestGeneratorsGi(t *testing.T) {
	wants := []string{
		"e68528ab16b201331fc980c33eef08f7d114554715d370a2c614182ef296dab3",
		"32011e4f5c29bbc20d5c96500e87e2303a004687895b2d6d944ff687d0dbefad",
		"0d688b311df06d633ced925c1561bea9608f305781c1ab32c55944628181cd1e",
		"e17522742ed8bd11aa5d1f2e341400eb1c6f85b47c46817ea0e90b5d5510b420",
		"67454d0f02d3962508b89d4209996943825dbf261e7e6e07a842d45b33b2baad",
		"c7f0c5eebcb5f37194b7ab96af66e79e0aa37a6cdbde5fbd6af13637b6f05cab",
		"c572d7c6f3ef692efbd13928dad208c4572ffe371d88f70a763af3a11cac8709",
		"e450cc93f07e0c18a79c1f0572a6971da37bfa81c6003835acf68a8afc1ca33b",
		"409ae3e34c0ff3929bceaf7b934923809b461038a1d31c7a0928c8c7ab707604",
		"c43d0400219b6745b95ff81176dfbbd5d33b9cc869e171411fff96656273b96c",
		"d1eeee54b75cc277bf8a6454accce6086ab24750b0d58a11fb7cad35eba42ff6",
		"2446b2efa69fb26a4268037909c466c9b5083bfecf3c2ab3a420114a6f91f0eb",
		"d0c4ee744ac129d0282a1554ca7a339e3d9db740826d365eefe984c0e5023969",
		"e1d621717a876830e0c7c1bf8e7e674cf5cbe3aa1e71885d7d347854277aa6ca",
		"6e95425b9481a70aa553f1e7d30b8182ef659f94ec4e446bc61058615460cbcc",
		"4200e80a3976d66f392c7998aa889e6a9efdc65abb6d39575ee8fd8b295008ad",
		"3e3e626d2c051c82de750847ced96e1f6af5f4a706703512914c0e334c3cf76e",
		"b98d0b73da8ae83754bc61c550c2c6ad76f78ba66e601c3876aea38e086552ae",
		"90128059cb3b5baa3b1230e2ef211257253d477490e182bcb60c89bae43752fe",
		"b04be209278413859ad51cf6d4a7f15bc2dea9f71c34f71945469705c3885b27",
		"fda85012a00938e6f12f4da3cb1642cd1963295d3b089dcb0ee81e73e1b14050",
		"73f1392e664fa1687983fcb1c7397b89876f6da8357ee8b07cb44534bc160644",
		"0f347deffff466dec1af40197d39e97933112af29d6f305734dc7a4c6e2aceaf",
		"c9c779f2644195546a17991a455a6d16a446305f80605e8466f5cd0861a6cb48",
		"56614c7cbd1f4b27100d84bd76b4e472237e09ad0970745da252ef0b197291b1",
		"4b266eaac77da3229fd884b4fc8163d8fae10a914334805a80b93da1ea8cb7ab",
		"e1b33961996a81b591fd54b72b67fe23c3bfac82223713865a39e9802c8a393e",
		"f1a19594ea8a6caa753c03d3e63a545ad8dc5ee331647bfeb7a9ac5b21cc04d8",
		"60f79007f42376ed140fe7efd43218106613546d8cb3bd06a5cef2e73b02fad7",
		"e9cb7b6fd3bb865dac6cff479bc2e3ce98ab95e4a6a57d81ae6d6cb032375f4a",
		"7ee2183153687344e093278bc692c4915761ada87a51a778b605e88078d9902a",
	}
	for i := 0; i < len(wants); i++ {
		want := fromHex(wants[i])
		p := Gi[i]
		got := p.bytes()
		if !constTimeEqual(want, got) {
			t.Errorf("G%d is encoded as %x, but must be %x", i, got, want)
		}
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
