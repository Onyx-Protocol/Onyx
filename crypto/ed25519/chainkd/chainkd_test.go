package chainkd

import (
	"reflect"
	"testing"
)

func TestChildKeys(t *testing.T) {
	rootXPrv, err := NewXPrv(nil)
	if err != nil {
		t.Fatal(err)
	}
	rootXPub := rootXPrv.XPub()

	const msg = "In the face of ignorance and resistance I wrote financial systems into existence"

	sig := rootXPrv.Sign([]byte(msg))
	if !rootXPub.Verify([]byte(msg), sig) {
		t.Error("root xpub cannot validate signature from root xprv")
	}

	sel := []byte{1, 2, 3}
	dprv := rootXPrv.Child(sel, false)
	dpub := rootXPub.Child(sel)
	t.Logf("* dpub via derivation: %s", dpub)

	sig = dprv.Sign([]byte(msg))
	if !dpub.Verify([]byte(msg), sig) {
		t.Error("derived xpub cannot validate sig from derived xprv [1]")
	}

	dpub = dprv.XPub()
	t.Logf("* dpub via extraction: %s", dpub)

	if !dpub.Verify([]byte(msg), sig) {
		t.Error("derived xpub cannot validate sig from derived xprv [2]")
	}

	dprv = dprv.Child(sel, false)
	sig = dprv.Sign([]byte(msg))
	dpub = dpub.Child(sel)
	if !dpub.Verify([]byte(msg), sig) {
		t.Error("double-derived xpub cannot validate sig from double-derived xprv")
	}

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
		sig := dprv.Sign([]byte(msg))
		if !dpub1.Verify([]byte(msg), sig) {
			t.Errorf("derived pubkey (%d) cannot validate signature from derived privkey", i)
		}

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
			sig = ddprv.Sign([]byte(msg))
			if !ddpub1.Verify([]byte(msg), sig) {
				t.Errorf("double-derived pubkey (%d, %d) cannot validate signature from double-derived privkey", i, j)
			}
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
		sig = hdprv.Sign([]byte(msg))
		if !hdpub.Verify([]byte(msg), sig) {
			t.Errorf("pubkey of hardened child %d cannot validate signature", i)
		}
	}
}
