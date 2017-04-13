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
		"6023c8e7633a9353a59bd930ea6dc397e400b1088b86b4a15d8de8567554df5574274bc1a0bd93b4494cb68e45c5ec5aefc1eed4d0c3bfd53b0b4e679ce52028")
	verifyTestVector(t, "Root(010203)/010203(H).xpub", root.Child([]byte{0x01, 0x02, 0x03}, true).XPub().hex(),
		"eabebab4184c63f8df07efe31fb588a0ae222318087458b4936bf0b0feab015074274bc1a0bd93b4494cb68e45c5ec5aefc1eed4d0c3bfd53b0b4e679ce52028")

	verifyTestVector(t, "Root(010203)/010203(N).xprv", root.Child([]byte{0x01, 0x02, 0x03}, false).hex(),
		"705afd25a0e242b7333105d77cbb0ec15e667154916bbed5084c355dba7b0748b0faca523928f42e685ee6deb0cb3d41a09617783c87e9a161a04f2207ad4d2f")
	verifyTestVector(t, "Root(010203)/010203(N).xpub", root.Child([]byte{0x01, 0x02, 0x03}, false).XPub().hex(),
		"c0bbd87142e7bf90abfbb3d0cccc210c6d7eb3f912c35f205302c86ae9ef6eefb0faca523928f42e685ee6deb0cb3d41a09617783c87e9a161a04f2207ad4d2f")
	verifyTestVector(t, "Root(010203)/010203(N).xpub", root.XPub().Child([]byte{0x01, 0x02, 0x03}).hex(),
		"c0bbd87142e7bf90abfbb3d0cccc210c6d7eb3f912c35f205302c86ae9ef6eefb0faca523928f42e685ee6deb0cb3d41a09617783c87e9a161a04f2207ad4d2f")

	verifyTestVector(t, "Root(010203)/010203(H)/“”(N).xprv", root.Child([]byte{0x01, 0x02, 0x03}, true).Child([]byte{}, false).hex(),
		"7023f9877813348ca8e67b29d551baf98a43cfb76cdff538f3ff97074a55df5560e3aa7fb600f61a84317a981dc9d1f7e8df2e8a3f8b544a21d2404e0b4e480a")
	verifyTestVector(t, "Root(010203)/010203(H)/“”(N).xpub", root.Child([]byte{0x01, 0x02, 0x03}, true).Child([]byte{}, false).XPub().hex(),
		"4e44c9ab8a45b9d1c3daab5c09d73b01209220ea704808f04feaa3614c7c7ba760e3aa7fb600f61a84317a981dc9d1f7e8df2e8a3f8b544a21d2404e0b4e480a")
	verifyTestVector(t, "Root(010203)/010203(H)/“”(N).xpub", root.Child([]byte{0x01, 0x02, 0x03}, true).XPub().Child([]byte{}).hex(),
		"4e44c9ab8a45b9d1c3daab5c09d73b01209220ea704808f04feaa3614c7c7ba760e3aa7fb600f61a84317a981dc9d1f7e8df2e8a3f8b544a21d2404e0b4e480a")

	verifyTestVector(t, "Root(010203)/010203(N)/“”(H).xprv", root.Child([]byte{0x01, 0x02, 0x03}, false).Child([]byte{}, true).hex(),
		"90b60b007e866dacc4b1f844089a805ffd78a295f5b0544034116ace354c58523410b1e6a3c557ca90c322f6ff4b5e547242965eaed8c34767765f0e05ed0e4f")
	verifyTestVector(t, "Root(010203)/010203(N)/“”(H).xpub", root.Child([]byte{0x01, 0x02, 0x03}, false).Child([]byte{}, true).XPub().hex(),
		"ca97ec34ef30aa08ebd19b9848b11ebadf9c0ad3a0be6b11d33d9558573aca633410b1e6a3c557ca90c322f6ff4b5e547242965eaed8c34767765f0e05ed0e4f")

	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xprv", root.Child([]byte{0x01, 0x02, 0x03}, false).Child([]byte{}, false).hex(),
		"d81ba3ab554a7d09bfd8bda5089363399b7f4b19d4f1806ca0c35feabf7b074856648f55e21bec3aa5df0bce0236aea88a4cc5c395c896df63676f095154bb7b")
	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xpub", root.Child([]byte{0x01, 0x02, 0x03}, false).Child([]byte{}, false).XPub().hex(),
		"28279bcb06aee9e5c0302f4e1db879ac7f5444ec07266a736dd571c21961427b56648f55e21bec3aa5df0bce0236aea88a4cc5c395c896df63676f095154bb7b")
	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xpub", root.XPub().Child([]byte{0x01, 0x02, 0x03}).Child([]byte{}).hex(),
		"28279bcb06aee9e5c0302f4e1db879ac7f5444ec07266a736dd571c21961427b56648f55e21bec3aa5df0bce0236aea88a4cc5c395c896df63676f095154bb7b")

	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xprv", root.Derive([][]byte{[]byte{0x01, 0x02, 0x03}, []byte{}}).hex(),
		"d81ba3ab554a7d09bfd8bda5089363399b7f4b19d4f1806ca0c35feabf7b074856648f55e21bec3aa5df0bce0236aea88a4cc5c395c896df63676f095154bb7b")
	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xprv", root.Derive([][]byte{[]byte{0x01, 0x02, 0x03}, []byte{}}).XPub().hex(),
		"28279bcb06aee9e5c0302f4e1db879ac7f5444ec07266a736dd571c21961427b56648f55e21bec3aa5df0bce0236aea88a4cc5c395c896df63676f095154bb7b")
	verifyTestVector(t, "Root(010203)/010203(N)/“”(N).xpub", root.XPub().Derive([][]byte{[]byte{0x01, 0x02, 0x03}, []byte{}}).hex(),
		"28279bcb06aee9e5c0302f4e1db879ac7f5444ec07266a736dd571c21961427b56648f55e21bec3aa5df0bce0236aea88a4cc5c395c896df63676f095154bb7b")
}

func TestVectors2(t *testing.T) {
	seed, _ := hex.DecodeString("fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542")
	root := RootXPrv(seed)

	verifyTestVector(t, "Root(fffcf9...).xprv", root.hex(),
		"0031615bdf7906a19360f08029354d12eaaedc9046806aefd672e3b93b024e495a95ba63cf47903eb742cd1843a5252118f24c0c496e9213bd42de70f649a798")
	verifyTestVector(t, "Root(fffcf9...).xpub", root.XPub().hex(),
		"f153ef65bbfaec3c8fd4fceb0510529048094093cf7c14970013282973e117545a95ba63cf47903eb742cd1843a5252118f24c0c496e9213bd42de70f649a798")

	verifyTestVector(t, "Root(fffcf9...)/0(N).xprv", root.Child([]byte{0x00}, false).hex(),
		"883e65e6e86499bdd170c14d67e62359dd020dd63056a75ff75983a682024e49e8cc52d8e74c5dfd75b0b326c8c97ca7397b7f954ad0b655b8848bfac666f09f")
	verifyTestVector(t, "Root(fffcf9...)/0(N).xpub", root.Child([]byte{0x00}, false).XPub().hex(),
		"f48b7e641d119b8ddeaf97aca104ee6e6a780ab550d40534005443550ef7e7d8e8cc52d8e74c5dfd75b0b326c8c97ca7397b7f954ad0b655b8848bfac666f09f")
	verifyTestVector(t, "Root(fffcf9...)/0(N).xpub", root.XPub().Child([]byte{0x00}).hex(),
		"f48b7e641d119b8ddeaf97aca104ee6e6a780ab550d40534005443550ef7e7d8e8cc52d8e74c5dfd75b0b326c8c97ca7397b7f954ad0b655b8848bfac666f09f")

	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H).xprv", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).hex(),
		"5048fa4498bf65e2b10d26e6c99cc43556ecfebf8b9fddf8bd2150ba29d63154044ef557a3aa4cb6ae8b61e87cb977a929bc4a170e4faafc2661231f5f3f78e8")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).XPub().hex(),
		"a8555c5ee5054ad03c6c6661968d66768fa081103bf576ea63a26c00ca7eab69044ef557a3aa4cb6ae8b61e87cb977a929bc4a170e4faafc2661231f5f3f78e8")

	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N).xprv", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).hex(),
		"480f6aa25f7c9f4a569896f06614303a697f00ee8d240c6277605d44e0d63154174c386ad6ae01e54acd7bb422243c6055058f4231e250050134283a76de8eff")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).XPub().hex(),
		"7385ab0b06eacc226c8035bab1ff9bc6972c7700d1caede26fe2b4d57b208bd0174c386ad6ae01e54acd7bb422243c6055058f4231e250050134283a76de8eff")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).XPub().Child([]byte{0x01}).hex(),
		"7385ab0b06eacc226c8035bab1ff9bc6972c7700d1caede26fe2b4d57b208bd0174c386ad6ae01e54acd7bb422243c6055058f4231e250050134283a76de8eff")

	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N)/2147483646(H).xprv", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).Child([]byte{0xfe, 0xff, 0xff, 0x7f}, true).hex(),
		"386014c6dfeb8dadf62f0e5acacfbf7965d5746c8b9011df155a31df7be0fb59986c923d979d89310acd82171dbaa7b73b20b2033ac6819d7f309212ff3fbabd")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N)/2147483646(H).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).Child([]byte{0xfe, 0xff, 0xff, 0x7f}, true).XPub().hex(),
		"9f66aa8019427a825dd72a13ce982454d99f221c8d4874db59f52c2945cbcabd986c923d979d89310acd82171dbaa7b73b20b2033ac6819d7f309212ff3fbabd")

	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N)/2147483646(H)/2(N).xprv", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).Child([]byte{0xfe, 0xff, 0xff, 0x7f}, true).Child([]byte{0x02}, false).hex(),
		"08c3772f5c0eee42f40d00f4faff9e4c84e5db3c4e7f28ecb446945a1de1fb59ef9d0a352f3252ea673e8b6bd31ac97218e019e845bdc545c268cd52f7af3f5d")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N)/2147483646(H)/2(N).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).Child([]byte{0xfe, 0xff, 0xff, 0x7f}, true).Child([]byte{0x02}, false).XPub().hex(),
		"67388f59a7b62644c3c6148575770e56969d77244530263bc9659b8563d7ff81ef9d0a352f3252ea673e8b6bd31ac97218e019e845bdc545c268cd52f7af3f5d")
	verifyTestVector(t, "Root(fffcf9...)/0(N)/2147483647(H)/1(N)/2147483646(H)/2(N).xpub", root.Child([]byte{0x00}, false).Child([]byte{0xff, 0xff, 0xff, 0x7f}, true).Child([]byte{0x01}, false).Child([]byte{0xfe, 0xff, 0xff, 0x7f}, true).XPub().Child([]byte{0x02}).hex(),
		"67388f59a7b62644c3c6148575770e56969d77244530263bc9659b8563d7ff81ef9d0a352f3252ea673e8b6bd31ac97218e019e845bdc545c268cd52f7af3f5d")
}

func TestExpandedPrivateKey(t *testing.T) {
	root := RootXPrv([]byte{0xca, 0xfe})
	verifyTestVector(t, "Root(cafe).xprv", root.hex(),
		"a0cde08fd2ea06e16dd5d21e64ca0609fa1d719b79fed4245a5b8ada0242464cebbc2b9e1e989aca72d9766efd9b63ebcfc968027ef27cb786babb7897f9248a")
	verifyTestVector(t, "Root(cafe).xprv.expandedkey", root.ExpandedPrivateKey().hex(),
		"a0cde08fd2ea06e16dd5d21e64ca0609fa1d719b79fed4245a5b8ada0242464c1437c8234e21e43eb9c79df0ce370dc82d4c7a952ef317e716b0762146bb61a0")

	child := root.Child([]byte{0xbe, 0xef}, false)
	verifyTestVector(t, "Root(cafe)/beef.xprv", child.hex(),
		"684df1aa25e0425c48c76392f42abc87a359ef2a2328ad31e53318128242464cf85916f4261b03f71afa64ad4bc2be4f335f15e433e815b45bbd15fcc7d1a864")
	verifyTestVector(t, "Root(cafe)/beef.xprv.expandedkey", child.ExpandedPrivateKey().hex(),
		"684df1aa25e0425c48c76392f42abc87a359ef2a2328ad31e53318128242464c0abdda57709eff7e9c60e0d4199065a6941122566c0a30ffa3ce0449d0582278")
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

func (key ExpandedPrivateKey) hex() []byte {
	hexBytes := make([]byte, hex.EncodedLen(len(key[:])))
	hex.Encode(hexBytes, key[:])
	return hexBytes
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
