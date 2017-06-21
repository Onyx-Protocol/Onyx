package ca

import (
	"bytes"

	"golang.org/x/crypto/sha3"
)

type EncryptedPacket struct {
	ct    []byte
	nonce [8]byte
	mac   [24]byte
}

// EncryptPacket encrypts plaintext binary string pt into buffer `ep` that must have size 32 bytes larger than the plaintext.
//
// Inputs:
// 1. `pt`: plaintext, a binary string of arbitrary byte length `n`.
// 2. `ek`: the encryption/authentication key.
// 3. `seed`: a unique seed per encryption (optional, empty by default) — could be a counter, timestamp, or a random string.
func EncryptPacket(ek, seed, pt []byte) *EncryptedPacket {
	result := new(EncryptedPacket)

	n := len(pt)

	// 1. Calculate an 8-byte `nonce`:
	//         nonce = StreamHash("Packet.Nonce", {seed,ek,pt}, 8 bytes)
	nonceHash := streamHash("ChainCA.Packet.Nonce", seed, ek, pt)
	nonceHash.Read(result.nonce[:])

	// 2. Calculate a keystream, a sequence of `n` bytes:
	//         keystream = StreamHash("Packet.Keystream", {nonce,ek}, n bytes)
	keystream := streamHash("ChainCA.Packet.Keystream", result.nonce[:], ek)
	// 3. Encrypt the plaintext payload by XORing each byte of plaintext with the corresponding byte of the keystream:
	//         ct[i] = pt[i] XOR keystream[i]
	// 4. Calculate 24-byte MAC using `ek` as a key and `ct||nonce` as a message:
	//         mac = KMAC128(K=ek, X=ct||nonce, S="ChainCA.Packet.MAC", L=24 bytes)
	kmac := sha3.NewKMAC128(ek, 24, []byte("ChainCA.Packet.MAC"))
	result.ct = make([]byte, n)
	var ks [1]byte
	for i := 0; i < n; i++ {
		keystream.Read(ks[:])
		c := pt[i] ^ ks[0]
		result.ct[i] = c
		kmac.Write([]byte{c})
	}
	kmac.Write(result.nonce[:])
	kmac.Sum(result.mac[:0])

	// 5. Return an encrypted packet `ep`, a sequence of `n+32` bytes:
	//         ep = ct || nonce || mac
	return result
}

// DecryptPacket decrypts packet into buffer `pt` that must have size 32 bytes shorter than the packet.
// Inputs:
// 1. `ep`: encrypted, a binary string of arbitrary byte length `n+32`.
// 2. `ek`: the encryption/authentication key.
//
// Output: the plaintext `pt` of length `n`, or `nil`, if authentication failed.
func (ep *EncryptedPacket) Decrypt(ek []byte) ([]byte, bool) {
	// 3. Compute MAC for the ciphertext concatenated with the nonce:
	//         mac’ = KMAC128(K=ek, X=ct||nonce, S="ChainCA.Packet.MAC", L=24 bytes)
	kmac := sha3.NewKMAC128(ek, 24, []byte("ChainCA.Packet.MAC"))
	kmac.Write(ep.ct)
	kmac.Write(ep.nonce[:])
	var mac2 [24]byte
	kmac.Sum(mac2[:0])

	// 4. Compare in constant time `mac’ == mac`. If not equal, return `nil`.
	if !constTimeEqual(ep.mac[:], mac2[:]) {
		return nil, false
	}

	// 5. Calculate a keystream, a sequence of `n` bytes:
	//         keystream = StreamHash("Packet.Keystream", {nonce,ek}, n bytes)
	// 6. Decrypt the plaintext payload by XORing each byte of ciphertext with the corresponding byte of the keystream:
	//         pt[i] = ct[i] XOR keystream[i]
	n := len(ep.ct)
	pt := make([]byte, n)
	keystream := streamHash("ChainCA.Packet.Keystream", ep.nonce[:], ek)
	var ks [1]byte
	for i := 0; i < n; i++ {
		keystream.Read(ks[:])
		pt[i] = ep.ct[i] ^ ks[0]
	}
	return pt, true
}

func (ep *EncryptedPacket) fill(p [][32]byte) {
	b := new(bytes.Buffer)
	for i := 0; i < len(p)-1; i++ {
		b.Write(p[i][:])
	}
	ep.ct = b.Bytes()
	copy(ep.nonce[:], p[len(p)-1][:8])
	copy(ep.mac[:], p[len(p)-1][8:])
}
