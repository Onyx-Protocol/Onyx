package ca

import (
	"fmt"

	"golang.org/x/crypto/sha3"
)

// EncryptPacket encrypts plaintext binary string pt into buffer `ep` that must have size 32 bytes larger than the plaintext.
//
// Inputs:
// 1. `pt`: plaintext, a binary string of arbitrary byte length `n`.
// 2. `ek`: the encryption/authentication key.
// 3. `seed`: a unique seed per encryption (optional, empty by default) — could be a counter, timestamp, or a random string.
//
// Output: `ct`: ciphertext, a binary string of byte length `n+32`.
//
// `ep` must be of size len(pt)+32. ep and pt may point to the same
// memory, but must coincide at position 0, not merely overlap.
func EncryptPacket(
	ek []byte,
	seed []byte,
	pt []byte,
	ep []byte,
) []byte {
	n := len(pt)

	if len(ep) != (n + 32) {
		panic(fmt.Errorf("pt has len %d, ep must have len %d (but has len %d)", n, n+32, len(ep)))
	}

	// 1. Calculate an 8-byte `nonce`:
	//         nonce = StreamHash("Packet.Nonce", {seed,ek,pt}, 8 bytes)
	nonceHash := streamHash("ChainCA.Packet.Nonce", seed, ek, pt)
	nonce := ep[n : n+8]
	nonceHash.Read(nonce[:])

	// 2. Calculate a keystream, a sequence of `n` bytes:
	//         keystream = StreamHash("Packet.Keystream", {nonce,ek}, n bytes)
	keystream := streamHash("ChainCA.Packet.Keystream", nonce[:], ek)
	// 3. Encrypt the plaintext payload by XORing each byte of plaintext with the corresponding byte of the keystream:
	//         ct[i] = pt[i] XOR keystream[i]
	// 4. Calculate 24-byte MAC using `ek` as a key and `ct||nonce` as a message:
	//         mac = KMAC128(K=ek, X=ct||nonce, S="ChainCA.Packet.MAC", L=24 bytes)
	kmac := sha3.NewKMAC128(ek, 24, []byte("ChainCA.Packet.MAC"))
	var ks [1]byte
	for i := 0; i < n; i++ {
		keystream.Read(ks[:])
		ep[i] = pt[i] ^ ks[0]
		kmac.Write(ep[i : i+1])
	}
	kmac.Write(nonce)
	kmac.Sum(ep[n+8 : n+8])

	// 5. Return an encrypted packet `ep`, a sequence of `n+32` bytes:
	//         ep = ct || nonce || mac
	return ep
}

// DecryptPacket decrypts packet into buffer `pt` that must have size 32 bytes shorter than the packet.
// Inputs:
// 1. `ep`: encrypted, a binary string of arbitrary byte length `n+32`.
// 2. `ek`: the encryption/authentication key.
//
// Output: the plaintext `pt` of length `n`, or `nil`, if authentication failed.
func DecryptPacket(
	ek []byte,
	ep []byte,
	pt []byte,
) bool {

	// 1. Verify that `ep` is at least 32 bytes long, otherwise return `nil`.
	m := len(ep)
	if m < 32 {
		return false
	}
	n := m - 32
	if len(pt) != n {
		panic("Buffer for decrypted packet must have size len(ep)-32.")
	}

	// 2. Split ciphertext `ep` into raw ciphertext `ct`, 8-byte `nonce` and 24-byte `mac`:
	nonce := ep[n : n+8]
	mac1 := ep[n+8 : n+32]

	// 3. Compute MAC for the ciphertext concatenated with the nonce:
	//         mac’ = KMAC128(K=ek, X=ct||nonce, S="ChainCA.Packet.MAC", L=24 bytes)
	kmac := sha3.NewKMAC128(ek, 24, []byte("ChainCA.Packet.MAC"))
	kmac.Write(ep[0 : n+8])
	var mac2 [24]byte
	kmac.Sum(mac2[:0])

	// 4. Compare in constant time `mac’ == mac`. If not equal, return `nil`.
	if !constTimeEqual(mac1[:], mac2[:]) {
		return false
	}

	// 5. Calculate a keystream, a sequence of `n` bytes:
	//         keystream = StreamHash("Packet.Keystream", {nonce,ek}, n bytes)
	// 6. Decrypt the plaintext payload by XORing each byte of ciphertext with the corresponding byte of the keystream:
	//         pt[i] = ct[i] XOR keystream[i]
	keystream := streamHash("ChainCA.Packet.Keystream", nonce[:], ek)
	var ks [1]byte
	for i := 0; i < n; i++ {
		keystream.Read(ks[:])
		pt[i] = ep[i] ^ ks[0]
	}
	return true
}
