package chainkd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
)

func TestVectors1(t *testing.T) {
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

	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xprv", root.Derive([][]byte{[]byte{0x01, 0x02, 0x03}, []byte{}}).hex(),
		"484148c20a28b663bc71d72e5f84df77e11ae9ac128d450b311635e6cd7c0748e70c8fb4062f4e8b4829ab1788d4a2ca71e056044503d6adfa75b229fb03d877")
	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xprv", root.Derive([][]byte{[]byte{0x01, 0x02, 0x03}, []byte{}}).XPub().hex(),
		"5786f826e7e09d17d6c1928952653275d81ad5fba728a16b5d0b04bd1ed9ee35e70c8fb4062f4e8b4829ab1788d4a2ca71e056044503d6adfa75b229fb03d877")
	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xpub", root.XPub().Derive([][]byte{[]byte{0x01, 0x02, 0x03}, []byte{}}).hex(),
		"5786f826e7e09d17d6c1928952653275d81ad5fba728a16b5d0b04bd1ed9ee35e70c8fb4062f4e8b4829ab1788d4a2ca71e056044503d6adfa75b229fb03d877")
}

func TestVectors2(t *testing.T) {
	seed, _ := hex.DecodeString("fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542")
	root := RootXPrv(seed)

	verifyTestVector(t, "Root(fffcf9...).xprv", root.hex(),
		"0031615bdf7906a19360f08029354d12eaaedc9046806aefd672e3b93b024e495a95ba63cf47903eb742cd1843a5252118f24c0c496e9213bd42de70f649a798")
	verifyTestVector(t, "Root(fffcf9...).xpub", root.XPub().hex(),
		"f153ef65bbfaec3c8fd4fceb0510529048094093cf7c14970013282973e117545a95ba63cf47903eb742cd1843a5252118f24c0c496e9213bd42de70f649a798")

	verifyTestVector(t, "Root(fffcf9...)/0(N).xprv", root.Child([]byte{0x00}, false).hex(),
		"b0d0dcae67caf04caca70eae5da892f862ffda71a12eeef0c857250c7e024e49ccb779405d72758d7fd6a562a221bdd30c430424d0b6871bbb54dd070fbbe477")
	verifyTestVector(t, "Root(fffcf9...)/0(N).xpub", root.Child([]byte{0x00}, false).XPub().hex(),
		"ef490a29370687a02e1915ddd583e13210b37882befb4381cc9dc14a488309acccb779405d72758d7fd6a562a221bdd30c430424d0b6871bbb54dd070fbbe477")
	verifyTestVector(t, "Root(fffcf9...)/0(N).xpub", root.XPub().Child([]byte{0x00}).hex(),
		"ef490a29370687a02e1915ddd583e13210b37882befb4381cc9dc14a488309acccb779405d72758d7fd6a562a221bdd30c430424d0b6871bbb54dd070fbbe477")

	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H).xprv", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).hex(),
		"787b2712f7dc8674040b212d0ef171dd96aa5c0c3df0104cf2bae8b224d442541c19a82372e94016d103187267ce952a988f80a371b061e4493619a52025ff01")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).XPub().hex(),
		"d807d625e9b55c3c099a7d43853f80429146f6f02b53469c1182a6bd45836d021c19a82372e94016d103187267ce952a988f80a371b061e4493619a52025ff01")

	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N).xprv", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).hex(),
		"a87166b647dceaac5723eebfdb3fde90007b38a60fd9de3d771ba9ef9ed442542f1ad730181f2a2c1160437dfcd71004c1a8cced671e7c9b772e35e1d47e0ae0")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).XPub().hex(),
		"794473aa11f7148b634e94444332f3ccafbfcb43617bd2751fc113218b1999af2f1ad730181f2a2c1160437dfcd71004c1a8cced671e7c9b772e35e1d47e0ae0")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).XPub().Child([]byte{0x01}).hex(),
		"794473aa11f7148b634e94444332f3ccafbfcb43617bd2751fc113218b1999af2f1ad730181f2a2c1160437dfcd71004c1a8cced671e7c9b772e35e1d47e0ae0")

	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N)/2147483646(H).xprv", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).Child([]byte{0xfe, 0xff, 0xff, 0x7f}, true).hex(),
		"d8abfb1acd80058f7c2590d92b4a4fac6d031fc73ec0229e326921df97987644d246e0ccb91215621d68b74b418ebdaece8b52b0d13fb1e47922910d8c30493f")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N)/2147483646(H).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).Child([]byte{0xfe, 0xff, 0xff, 0x7f}, true).XPub().hex(),
		"11aa4807361528e2bca0f26914b570d84cee26f8603378aa4c36fd1b76ec78ead246e0ccb91215621d68b74b418ebdaece8b52b0d13fb1e47922910d8c30493f")

	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N)/2147483646(H)/2(N).xprv", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).Child([]byte{0xfe, 0xff, 0xff, 0x7f}, true).Child([]byte{0x02}, false).hex(),
		"281e8c4aff2fdeb0018ef283b64755f99baf993e3acccee4d87484f21f99764443388d06e53716a83060d4df1cdccfe8364029cfdb9422d5bffc31732fdca243")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N)/2147483646(H)/2(N).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).Child([]byte{0xfe, 0xff, 0xff, 0x7f}, true).Child([]byte{0x02}, false).XPub().hex(),
		"7d152694a55b166f7038aa4aee0f5865c8e777caec8778c41b95d8997754f80343388d06e53716a83060d4df1cdccfe8364029cfdb9422d5bffc31732fdca243")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N)/2147483646(H)/2(N).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).Child([]byte{0xfe, 0xff, 0xff, 0x7f}, true).XPub().Child([]byte{0x02}).hex(),
		"7d152694a55b166f7038aa4aee0f5865c8e777caec8778c41b95d8997754f80343388d06e53716a83060d4df1cdccfe8364029cfdb9422d5bffc31732fdca243")
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

func TestBits(t *testing.T) {
	for i := 0; i < 256; i++ {
		root := RootXPrv([]byte{byte(i)})

		rootbytes := root.Bytes()
		if rootbytes[0]&7 != 0 {
			t.Errorf("ChainKD root key must have low 3 bits set to '000'")
		}
		if (rootbytes[31] >> 5) != 2 {
			t.Errorf("ChainKD root key must have high 3 bits set to '010'")
		}

		xprv := root
		for d := 0; d < 1000; d++ { // at least after 1000 levels necessary bits are survived
			xprv = xprv.Child([]byte("child"), false)
			xprvbytes := xprv.Bytes()

			if xprvbytes[0]&7 != 0 {
				t.Errorf("ChainKD non-hardened child key must have low 3 bits set to '000'")
			}
			if xprvbytes[31]>>6 != 1 {
				t.Errorf("ChainKD non-hardened child key must have high 2 bits set to '10' (LE)")
			}

			hchild := xprv.Child([]byte("hardened child"), true)
			hchildbytes := hchild.Bytes()
			if hchildbytes[0]&7 != 0 {
				t.Errorf("ChainKD hardened key must have low 3 bits set to '000'")
			}
			if (hchildbytes[31] >> 5) != 2 {
				t.Errorf("ChainKD hardened key must have high 3 bits set to '010'")
			}
		}
	}
}
