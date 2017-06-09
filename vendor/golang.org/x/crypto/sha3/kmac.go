package sha3

import "hash"

type kmac struct { // implements hash.Hash and ShakeHash
	d             *state
	lengthEmitted bool
}

// NewKMAC128 creates an instance of Hash with a given key,
// output length in bytes and a customization string s.
func NewKMAC128(key []byte, length int, s []byte) hash.Hash {
	return newKMAC(128, key, length, s)
}

// NewKMAC256 creates an instance of Hash with a given key,
// output length in bytes and a customization string s.
func NewKMAC256(key []byte, length int, s []byte) hash.Hash {
	return newKMAC(256, key, length, s)
}

// NewKMACXOF128 provides an arbitrary-length output.
func NewKMACXOF128(key []byte, s []byte) ShakeHash {
	return newKMAC(128, key, 0, s)
}

// NewKMACXOF256 provides an arbitrary-length output.
func NewKMACXOF256(key []byte, s []byte) ShakeHash {
	return newKMAC(256, key, 0, s)
}

// BlockSize returns the rate of sponge underlying this hash function.
func (k *kmac) BlockSize() int { return k.d.rate }

// Size returns the output size of the hash function in bytes.
func (k *kmac) Size() int { return k.d.outputLen }

func (k *kmac) Reset() {
	k.lengthEmitted = false
	k.d.Reset()
}

func (k *kmac) Clone() ShakeHash {
	return k.clone()
}

func (k *kmac) Write(p []byte) (written int, err error) {
	return k.d.Write(p)
}

func (k *kmac) Read(out []byte) (n int, err error) {
	n = 0
	if !k.lengthEmitted {
		n = k.encodeOutputLength()
		k.lengthEmitted = true
	}
	m, err := k.d.Read(out)
	return n + m, err
}

// Sum applies padding to the hash state and then squeezes out the desired
// number of output bytes.
func (k *kmac) Sum(in []byte) []byte {
	// Make a copy of the original hash so that caller can keep writing
	// and summing.
	dup := k.clone()
	hash := make([]byte, dup.d.outputLen)
	dup.Read(hash)
	return append(in, hash...)
}

func newKMAC(securitybits int, key []byte, length int, s []byte) *kmac {
	k := kmac{d: newCShake(securitybits, []byte("KMAC"), s)}

	k.d.outputLen = length

	// bytepad(encode_string(K), 168):
	var n int
	n += leftEncode(k.d, uint64(k.d.rate))
	n += encodeString(k.d, key)
	k.d.Write(zero[:k.d.rate-(n%k.d.rate)])

	return &k
}

func (k *kmac) encodeOutputLength() int {
	return rightEncode(k.d, uint64(8*k.d.outputLen))
}

func (k *kmac) clone() *kmac {
	k2 := *k
	k2.d = k2.d.clone()
	return &k2
}
