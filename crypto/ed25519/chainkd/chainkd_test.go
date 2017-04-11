package chainkd

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func TestVectors(t *testing.T) {
	root := RootXPrv([]byte{0x01, 0x02, 0x03})

	verifyTestVector(t, "Root(010203).xprv", root.hex(),
		"50f8c532ce6f088de65c2c1fbc27b491509373fab356eba300dfa7cc587b07483bc9e0d93228549c6888d3f68ad664b92c38f5ea8ca07181c1410949c02d3146")
	verifyTestVector(t, "Root(010203).xpub", root.XPub().hex(),
		"e11f321ffef364d01c2df2389e61091b15dab2e8eee87cb4c053fa65ed2812993bc9e0d93228549c6888d3f68ad664b92c38f5ea8ca07181c1410949c02d3146")

	verifyTestVector(t, "Root(010203)/010203(H).xprv", root.Child([]byte{0x01, 0x02, 0x03}, true).hex(),
		"98bd4126e9040d7dfcf6c4d1ceb674db0569e7f21266eebf3dc9a469bab1ab45200bd2d6a956e819c68134a40be13e2653ccdcbaab92f7fd492626886884f832")
	verifyTestVector(t, "Root(010203)/010203(H).xpub", root.Child([]byte{0x01, 0x02, 0x03}, true).XPub().hex(),
		"696809f6ac24c8b70dde8778a8a0db26f642388be12b6323f12a97fcc3cbccbb200bd2d6a956e819c68134a40be13e2653ccdcbaab92f7fd492626886884f832")

	verifyTestVector(t, "Root(010203)/010203(N).xprv", root.Child([]byte{0x01, 0x02, 0x03}, false).hex(),
		"30837f155673a659f5c659045b598b175ceea3724c07dc53910e8392df7b0748d40ba49ebee85271fd1d53a45bfbb228623e98c43227fd1484f17139736f2f39")
	verifyTestVector(t, "Root(010203)/010203(N).xpub", root.Child([]byte{0x01, 0x02, 0x03}, false).XPub().hex(),
		"2e457bd3bd135cbe5bd46821588ad82b74e8b9cb256e3a956d72322df61b51acd40ba49ebee85271fd1d53a45bfbb228623e98c43227fd1484f17139736f2f39")
	verifyTestVector(t, "Root(010203)/010203(N).xpub", root.XPub().Child([]byte{0x01, 0x02, 0x03}).hex(),
		"2e457bd3bd135cbe5bd46821588ad82b74e8b9cb256e3a956d72322df61b51acd40ba49ebee85271fd1d53a45bfbb228623e98c43227fd1484f17139736f2f39")

	verifyTestVector(t, "Root(010203)/010203(H)/“”(N).xprv", root.Child([]byte{0x01, 0x02, 0x03}, true).Child([]byte{}, false).hex(),
		"0889925d37b9664af32c78cb8225022b5854586c08f8a9a7ed3a657279b2ab45ae8c6d29a2d80e7dc8a141058ff68c257e59c45daba3184b100456828ed9ade8")
	verifyTestVector(t, "Root(010203)/010203(H)/“”(N).xpub", root.Child([]byte{0x01, 0x02, 0x03}, true).Child([]byte{}, false).XPub().hex(),
		"6b45415a0638feb47a5eab07961883fafe476b637de7004111317a2454465ae2ae8c6d29a2d80e7dc8a141058ff68c257e59c45daba3184b100456828ed9ade8")
	verifyTestVector(t, "Root(010203)/010203(H)/“”(N).xpub", root.Child([]byte{0x01, 0x02, 0x03}, true).XPub().Child([]byte{}).hex(),
		"6b45415a0638feb47a5eab07961883fafe476b637de7004111317a2454465ae2ae8c6d29a2d80e7dc8a141058ff68c257e59c45daba3184b100456828ed9ade8")

	verifyTestVector(t, "Root(010203)/010203(N)/“”(H).xprv", root.Child([]byte{0x01, 0x02, 0x03}, false).Child([]byte{}, true).hex(),
		"b8b626e7ce7e86c7e673e5652de643b98631771bb1602136bdb154863e606e5c360b2aee72cb1b1d62eccba447c164629ea956758982ccbb0a1a26fc991b7fd2")
	verifyTestVector(t, "Root(010203)/010203(N)/“”(H).xpub", root.Child([]byte{0x01, 0x02, 0x03}, false).Child([]byte{}, true).XPub().hex(),
		"174eba73de14f9af2693c63c16e3466577ffc4e780846c8ff81f69fd0346af83360b2aee72cb1b1d62eccba447c164629ea956758982ccbb0a1a26fc991b7fd2")

	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xprv", root.Child([]byte{0x01, 0x02, 0x03}, false).Child([]byte{}, false).hex(),
		"484148c20a28b663bc71d72e5f84df77e11ae9ac128d450b311635e6cd7c0748e70c8fb4062f4e8b4829ab1788d4a2ca71e056044503d6adfa75b229fb03d877")
	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xpub", root.Child([]byte{0x01, 0x02, 0x03}, false).Child([]byte{}, false).XPub().hex(),
		"5786f826e7e09d17d6c1928952653275d81ad5fba728a16b5d0b04bd1ed9ee35e70c8fb4062f4e8b4829ab1788d4a2ca71e056044503d6adfa75b229fb03d877")
	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xpub", root.XPub().Child([]byte{0x01, 0x02, 0x03}).Child([]byte{}).hex(),
		"5786f826e7e09d17d6c1928952653275d81ad5fba728a16b5d0b04bd1ed9ee35e70c8fb4062f4e8b4829ab1788d4a2ca71e056044503d6adfa75b229fb03d877")


	

	// TBD: more test vectors
}

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
		for mask := byte(1); mask != 0; mask <<= 1 {
			xpub.data[i] ^= mask
			if xpub.Verify(msg, sig) {
				t.Fatalf("altered %s should not verify signature from %s", xpubdesc, xprvdesc)
			}
			xpub.data[i] ^= mask
		}
	}

	// permute only 1/7th of the bits to make tests run faster
	for i := 0; i < len(msg); i += 7 {
		for mask := byte(1); mask != 0; mask <<= 1 {
			msg[i] ^= mask
			if xpub.Verify(msg, sig) {
				t.Fatalf("%s should not verify signature from %s against altered message", xpubdesc, xprvdesc)
			}
			msg[i] ^= mask
		}
	}

	for i := 0; i < len(sig); i++ {
		for mask := byte(1); mask != 0; mask <<= 1 {
			sig[i] ^= mask
			if xpub.Verify(msg, sig) {
				t.Fatalf("%s should not verify altered signature from %s", xpubdesc, xprvdesc)
			}
			sig[i] ^= mask
		}
	}
}

func verifyTestVector(t *testing.T, message string, got []byte, want string) {
	if !bytes.Equal(got, []byte(want)) {
		t.Errorf("ChainKD Test Vector: %s:\n    got  = %s\n    want = %s", message, got, want)
	}
}

func (xpub XPub) hex() []byte {
	s, _ := xpub.MarshalText()
	return s
}

func (xprv XPrv) hex() []byte {
	s, _ := xprv.MarshalText()
	return s
}

func TestEdDSABits(t *testing.T) {
	// TBD: make sure that even after 2^20 derivations the low 3 bits and the high 2 bits are stable.
	
}
