package chainkd

import (
	"fmt"
	"reflect"
	"testing"
)

func TestChildKeys(t *testing.T) {
	rootXPrv, err := NewXPrv(nil)
	if err != nil {
		t.Fatal(err)
	}
	rootXPub := rootXPrv.XPub()

	msg := []byte("In the face of ignorance and resistance I wrote financial systems into existence")

	sig := rootXPrv.Sign(msg)
	doverify(t, rootXPub, msg, sig, "root xpub", "root xprv")

	sel := []byte{1, 2, 3}
	dprv := rootXPrv.Child(sel, false)
	dpub := rootXPub.Child(sel)

	sig = dprv.Sign(msg)
	doverify(t, dpub, msg, sig, "derived xpub", "derived xprv")

	dpub = dprv.XPub()
	doverify(t, dpub, msg, sig, "xpub from derived xprv", "derived xprv")

	dprv = dprv.Child(sel, false)
	sig = dprv.Sign(msg)
	dpub = dpub.Child(sel)
	doverify(t, dpub, msg, sig, "double-derived xpub", "double-derived xprv")

	for i := byte(0); i < 10; i++ {
		sel := []byte{i}

		// Non-hardened children
		dprv := rootXPrv.Child(sel, false)
		if reflect.DeepEqual(dprv, rootXPrv) {
			t.Errorf("derived private key %d is the same as the root", i)
		}
		dpub1 := rootXPub.Child(sel)
		if reflect.DeepEqual(dpub1, rootXPub) {
			t.Errorf("derived public key %d is the same as the root", i)
		}
		sig := dprv.Sign(msg)
		doverify(t, dpub1, msg, sig, fmt.Sprintf("derived pubkey (%d)", i), "derived xprv")

		for j := byte(0); j < 10; j++ {
			sel2 := []byte{j}
			ddprv := dprv.Child(sel2, false)
			if reflect.DeepEqual(ddprv, dprv) {
				t.Errorf("rootXPrv.Child(%d).Child(%d) is the same as its parent", i, j)
			}
			ddpub1 := dpub1.Child(sel2)
			if reflect.DeepEqual(ddpub1, dpub1) {
				t.Errorf("rootXPub.Child(%d).Child(%d) is the same as its parent", i, j)
			}
			sig = ddprv.Sign(msg)
			doverify(t, ddpub1, msg, sig, fmt.Sprintf("double-derived pubkey (%d, %d)", i, j), "double-derived xprv")
		}

		// Hardened children
		hdprv := rootXPrv.Child(sel, true)
		if reflect.DeepEqual(hdprv, rootXPrv) {
			t.Errorf("derived hardened privkey %d is the same as the root", i)
		}
		if reflect.DeepEqual(hdprv, dprv) {
			t.Errorf("derived hardened privkey %d is the same as the unhardened derived privkey", i)
		}
		hdpub := hdprv.XPub()
		if reflect.DeepEqual(hdpub, dpub1) {
			t.Errorf("pubkey of hardened child %d is the same as pubkey of non-hardened child", i)
		}
		sig = hdprv.Sign(msg)
		doverify(t, hdpub, msg, sig, fmt.Sprintf("pubkey of hardened child %d", i), "derived xprv")
	}
}

func doverify(t *testing.T, xpub XPub, msg, sig []byte, xpubdesc, xprvdesc string) {
	if !xpub.Verify(msg, sig) {
		t.Errorf("%s cannot verify signature from %s", xpubdesc, xprvdesc)
	}

	for i := 0; i < 32; i++ {
		xpub[i] ^= 0xff
		if xpub.Verify(msg, sig) {
			t.Fatalf("altered %s should not verify signature from %s", xpubdesc, xprvdesc)
		}
		xpub[i] ^= 0xff
	}

	for i := 0; i < len(msg); i++ {
		msg[i] ^= 0xff
		if xpub.Verify(msg, sig) {
			t.Fatalf("%s should not verify signature from %s against altered message", xpubdesc, xprvdesc)
		}
		msg[i] ^= 0xff
	}

	for i := 0; i < len(sig); i++ {
		sig[i] ^= 0xff
		if xpub.Verify(msg, sig) {
			t.Fatalf("%s should not verify altered signature from %s", xpubdesc, xprvdesc)
		}
		sig[i] ^= 0xff
	}
}
