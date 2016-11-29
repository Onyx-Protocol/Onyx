package ca

import (
	"bytes"
	"testing"
)

func Test0PartyRingSignature(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))
	aliceKey := reducedScalar(hash512([]byte("alice")))
	pubkeys := []Point{}

	ringsig := createRingSignature(msg, pubkeys, 0, aliceKey)

	if err := ringsig.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature is not verified correctly: %s", err)
	}

	want := fromHex("0000000000000000000000000000000000000000000000000000000000000000")
	var b bytes.Buffer
	err := ringsig.writeTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), want) {
		t.Errorf("Got %x, want %x", b.Bytes(), want)
	}
}

func Test1PartyRingSignature(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))
	aliceKey := reducedScalar(hash512([]byte("alice")))
	alicePubkey := multiplyBasePoint(aliceKey)
	pubkeys := []Point{
		alicePubkey,
	}

	ringsig := createRingSignature(msg, pubkeys, 0, aliceKey)

	if err := ringsig.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature is not verified correctly: %s", err)
	}

	want := fromHex("" +
		"fa0a00c10f5a7ebabaf71f72091dcd9443d6b74a1225ea68cacf5631ba734301" +
		"894259d2ada654559a9193b1c041c985f0ea47e6b49401399ab69e3479a9da92")
	var b bytes.Buffer
	err := ringsig.writeTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), want) {
		t.Errorf("Got %x, want %x", b.Bytes(), want)
	}
}

func Test2PartyRingSignature(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))
	aliceKey := reducedScalar(hash512([]byte("alice")))
	bobKey := reducedScalar(hash512([]byte("bob")))
	alicePubkey := multiplyBasePoint(aliceKey)
	bobPubkey := multiplyBasePoint(bobKey)
	pubkeys := []Point{
		alicePubkey,
		bobPubkey,
	}

	ringsig1 := createRingSignature(msg, pubkeys, 0, aliceKey)
	ringsig2 := createRingSignature(msg, pubkeys, 1, bobKey)

	if err := ringsig1.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature 1 is not verified correctly: %s", err)
	}
	if err := ringsig2.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature 2 is not verified correctly: %s", err)
	}

	want1 := fromHex("" +
		"4667a78a13971cc6ad21191e6727db0d061401c1f9298297dc2258716a266e09" +
		"8549562a605878bbb53bc3ae1792fe590095ade4c391aefbb45fe31202a9029b" +
		"3c36493211a33c13b31669d741c012514383f69d9f1122be522c713bd1dd1d0e")
	var b bytes.Buffer
	err := ringsig1.writeTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), want1) {
		t.Errorf("Got %x, want %x", b.Bytes(), want1)
	}

	want2 := fromHex("" +
		"81d6197cb24636ba92a81ab649efa6884b771fee96d96a11963519d1c3efaf08" +
		"e6b9fe11ac1c5433f94dbcf0a9872ad245a57eaa2fdde0981ad3fc1fbb8a489a" +
		"5f160ce5789d9c70a5973e57a1bd4a89800c8b103f1a3f0c9c41abf1dcd2f25d")
	b.Reset()
	err = ringsig2.writeTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), want2) {
		t.Errorf("Got %x, want %x", b.Bytes(), want2)
	}
}

func Test3PartyRingSignature(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))

	aliceKey := reducedScalar(hash512([]byte("alice")))
	bobKey := reducedScalar(hash512([]byte("bob")))
	carlKey := reducedScalar(hash512([]byte("carl")))

	alicePubkey := multiplyBasePoint(aliceKey)
	bobPubkey := multiplyBasePoint(bobKey)
	carlPubkey := multiplyBasePoint(carlKey)

	pubkeys := []Point{
		alicePubkey,
		bobPubkey,
		carlPubkey,
	}

	ringsig1 := createRingSignature(msg, pubkeys, 0, aliceKey)
	ringsig2 := createRingSignature(msg, pubkeys, 1, bobKey)
	ringsig3 := createRingSignature(msg, pubkeys, 2, carlKey)

	if err := ringsig1.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature 1 is not verified correctly: %s", err)
	}

	if err := ringsig2.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature 2 is not verified correctly: %s", err)
	}

	if err := ringsig3.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature 3 is not verified correctly: %s", err)
	}

	want1 := fromHex("" +
		"d2da942b6957a0406468ddfef63837a30362e2bf147dd6999f26e10a7c66000c" +
		"684655f10fd04d91c286e0ad125ff057d7d06c33af75806e145e97c21aa0c7fe" +
		"805f7b62d2d2c5fc1c5b473774e8aaf6ca57109c594359c5788b20de2d65a451" +
		"9943d46be56f3231e0b465748c56836ed261ee45b157f59dd9f359ab6a61d757")
	var b bytes.Buffer
	err := ringsig1.writeTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), want1) {
		t.Errorf("Got %x, want %x", b.Bytes(), want1)
	}
	want2 := fromHex("" +
		"755c8ddec50106899c0eddede35345708fd5fbf1f72c8a994728112e7e989f0b" +
		"89f771d0d185441690110d5b915821baeb483e494d2dd55782822327863b6d1a" +
		"1ed96022b01b23883f611d341f2059116de465543d384092988ef55d3cffebe5" +
		"3212556130fbd7124282398074f179d65ed69356f6c031d94c560c3b2dcff975")
	b.Reset()
	err = ringsig2.writeTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), want2) {
		t.Errorf("Got %x, want %x", b.Bytes(), want2)
	}

	want3 := fromHex("" +
		"f3d5e3b6d181756454ba4e0a72aa1f8bb08907961a4cefe95d3177a2f8a7930e" +
		"6355e95f940192b7ccf9bb4c983a1782f767145655743753750f1ba212ad9410" +
		"3236c19443686c570a2fbcfd4d62b21a8c362d7a491a6f8106508d8f9aa2baac" +
		"8d436a8fcd7dcf2f446e2446b634226e014bcc3d0611913d359d860d4d2dca60")
	b.Reset()
	err = ringsig3.writeTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), want3) {
		t.Errorf("Got %x, want %x", b.Bytes(), want3)
	}
}

func Test4PartyRingSignature(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))

	aliceKey := reducedScalar(hash512([]byte("alice")))
	bobKey := reducedScalar(hash512([]byte("bob")))
	carlKey := reducedScalar(hash512([]byte("carl")))
	danKey := reducedScalar(hash512([]byte("dan")))

	alicePubkey := multiplyBasePoint(aliceKey)
	bobPubkey := multiplyBasePoint(bobKey)
	carlPubkey := multiplyBasePoint(carlKey)
	danPubkey := multiplyBasePoint(danKey)

	pubkeys := []Point{
		alicePubkey,
		bobPubkey,
		carlPubkey,
		danPubkey,
	}

	ringsig1 := createRingSignature(msg, pubkeys, 0, aliceKey)
	ringsig2 := createRingSignature(msg, pubkeys, 1, bobKey)
	ringsig3 := createRingSignature(msg, pubkeys, 2, carlKey)
	ringsig4 := createRingSignature(msg, pubkeys, 3, danKey)

	if err := ringsig1.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature is not verified correctly: %s", err)
	}
	if err := ringsig2.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature is not verified correctly: %s", err)
	}
	if err := ringsig3.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature is not verified correctly: %s", err)
	}
	if err := ringsig4.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature is not verified correctly: %s", err)
	}

	want1 := fromHex("" +
		"118d191bd27863dc3169aee7eb32c4e82292213553f40a70dfa12c957ddfac00" +
		"55479c540093ecfcd06859b954296bd6c81d5a79d582539d842013bf8fe6ec78" +
		"433965b2425b35c39b5e4564915fc934bd90af6918caf2ba19f8dfb4369d19d2" +
		"0984918eb5990e385ca044bad25970e4080547ccd36d3434a101c3bc9f36db07" +
		"e5beb08fd3b1a7b4f30799bc6a8672dd675f60af169387112025e8e8cbd7ba44")
	var b bytes.Buffer
	err := ringsig1.writeTo(&b)
	if !bytes.Equal(b.Bytes(), want1) {
		t.Errorf("Got %x, want %x", b.Bytes(), want1)
	}
	want2 := fromHex("" +
		"1d883622875424491f33ae3b9f19c5ebd2c666a1fa2f21f1cfe3102a12ba7e0f" +
		"dc23d1759fd9acf8d90edd3de27cbb86c9dcdc39835ff0a4aa82e2b257f1186a" +
		"4b5062def0a8f982d1d355c0138867c6190b4db74a97ccb987fd4a60962f3468" +
		"774a3564cc392eb6457f60d1de965ad8809e05a60e8decab98b525a52f8525b8" +
		"3454cae3cc6a855dfc8e880ec1dd49d98c5ef74ba02aaac239dc767446c0d1ce")
	b.Reset()
	err = ringsig2.writeTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), want2) {
		t.Errorf("Got %x, want %x", b.Bytes(), want2)
	}
}

func TestInvalidRingSignatures(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))

	aliceKey := reducedScalar(hash512([]byte("alice")))
	bobKey := reducedScalar(hash512([]byte("bob")))
	carlKey := reducedScalar(hash512([]byte("carl")))
	danKey := reducedScalar(hash512([]byte("dan")))

	alicePubkey := multiplyBasePoint(aliceKey)
	bobPubkey := multiplyBasePoint(bobKey)
	carlPubkey := multiplyBasePoint(carlKey)
	danPubkey := multiplyBasePoint(danKey)

	pubkeys := []Point{
		alicePubkey,
		bobPubkey,
		carlPubkey,
		danPubkey,
	}

	ringsig := createRingSignature(msg, pubkeys, 1, bobKey)

	if err := ringsig.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature is not verified correctly: %s", err)
	}

	for j := 0; j < 32; j++ {
		for bit := uint(0); bit < 8; bit++ {
			ringsig.e[j] ^= byte(1 << bit)
			if err := ringsig.verify(msg, pubkeys); err == nil {
				t.Errorf("unexpected success from RingSignature.Verify (flipped the %dth bit in ringsig.e)", j*8+int(bit))
			}
			ringsig.e[j] ^= (1 << bit)
		}
	}
	for i := range ringsig.s {
		for j := 0; j < 32; j++ {
			for bit := uint(0); bit < 8; bit++ {
				ringsig.s[i][j] ^= byte(1 << bit)
				if err := ringsig.verify(msg, pubkeys); err == nil {
					t.Errorf("unexpected success from RingSignature.Verify (flipped the %dth bit in ringsig.s[%d])", j*8+int(bit), i)
				}
				ringsig.s[i][j] ^= (1 << bit)
			}
		}
	}
}

func TestWrongPubkeysInRingSignature(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))

	aliceKey := reducedScalar(hash512([]byte("alice")))
	bobKey := reducedScalar(hash512([]byte("bob")))
	carlKey := reducedScalar(hash512([]byte("carl")))
	danKey := reducedScalar(hash512([]byte("dan")))

	alicePubkey := multiplyBasePoint(aliceKey)
	bobPubkey := multiplyBasePoint(bobKey)
	carlPubkey := multiplyBasePoint(carlKey)
	danPubkey := multiplyBasePoint(danKey)

	pubkeys := []Point{
		alicePubkey,
		bobPubkey,
		carlPubkey,
	}

	ringsig := createRingSignature(msg, pubkeys, 1, bobKey)
	if err := ringsig.verify(msg, pubkeys); err != nil {
		t.Errorf("Ring signature should be valid: %s", err)
	}

	ringsig = createRingSignature(msg, pubkeys, 1, bobKey)
	msg2 := msg
	msg2[0] ^= 1
	if err := ringsig.verify(msg2, pubkeys); err == nil {
		t.Errorf("Ring signature should not be valid for another message")
	}

	ringsig = createRingSignature(msg, pubkeys, 0, bobKey)
	if err := ringsig.verify(msg, pubkeys); err == nil {
		t.Errorf("Ring signature cannot be created with the wrong pubkey index")
	}

	ringsig = createRingSignature(msg, pubkeys, 1, bobKey)
	if err := ringsig.verify(msg, []Point{bobPubkey, alicePubkey, carlPubkey}); err == nil {
		t.Errorf("Ring signature cannot be verified with the wrong pubkey order")
	}

	ringsig = createRingSignature(msg, pubkeys, 0, bobKey)
	if err := ringsig.verify(msg, []Point{bobPubkey, alicePubkey, carlPubkey}); err == nil {
		t.Errorf("Ring signature cannot be verified with the wrong pubkey order even if incorrect index at signing matches the pubkey during verification.")
	}

	ringsig = createRingSignature(msg, pubkeys, 1, bobKey)
	if err := ringsig.verify(msg, []Point{alicePubkey, bobPubkey, carlPubkey, danPubkey}); err == nil {
		t.Errorf("Ring signature cannot be verified with the additional pubkeys.")
	}
}

func ringsigsEqual(rs1, rs2 *ringSignature) bool {
	if rs1.e != rs2.e {
		return false
	}
	if len(rs1.s) != len(rs2.s) {
		return false
	}
	for i, s := range rs1.s {
		if s != rs2.s[i] {
			return false
		}
	}
	return true
}
