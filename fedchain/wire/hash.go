package wire

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
)

// Hash20 is a RIPEMD-160(SHA256(data)) hash. Typically used for
// addresses and asset ids.
type Hash20 [20]byte

// NewHash20FromStr reads a base58 encoded string and returns a Hash20
func NewHash20FromStr(str string) (Hash20, error) {
	var hash Hash20
	res, _, err := base58.CheckDecode(str)
	copy(hash[:], res)
	return hash, err
}

// Hash32 is used in several of the bitcoin messages and common structures.  It
// typically represents the double sha256 of data.
type Hash32 [32]byte

// MaxHashStringSize is the maximum length of a Hash32 hash string.
const MaxHashStringSize = len(Hash32{}) * 2

// ErrHashStrSize describes an error that indicates the caller specified a hash
// string that has too many characters.
var ErrHashStrSize = fmt.Errorf("max hash string length is %v bytes", MaxHashStringSize)

// String returns the Hash32 as the hexadecimal string of the byte-reversed
// hash.
func (hash Hash32) String() string {
	for i := 0; i < len(Hash32{})/2; i++ {
		hash[i], hash[len(Hash32{})-1-i] = hash[len(Hash32{})-1-i], hash[i]
	}
	return hex.EncodeToString(hash[:])
}

// Bytes returns the bytes which represent the hash as a byte slice.
//
// NOTE: This makes a copy of the bytes and should have probably been named
// CloneBytes.  It is generally cheaper to just slice the hash directly thereby
// reusing the same bytes rather than calling this method.
func (hash *Hash32) Bytes() []byte {
	newHash := make([]byte, len(Hash32{}))
	copy(newHash, hash[:])

	return newHash
}

// SetBytes sets the bytes which represent the hash.  An error is returned if
// the number of bytes passed in is not len(Hash32{}).
func (hash *Hash32) SetBytes(newHash []byte) error {
	nhlen := len(newHash)
	if nhlen != len(Hash32{}) {
		return fmt.Errorf("invalid sha length of %v, want %v", nhlen,
			len(Hash32{}))
	}
	copy(hash[:], newHash)

	return nil
}

// IsEqual returns true if target is the same as hash.
func (hash *Hash32) IsEqual(target *Hash32) bool {
	return *hash == *target
}

// NewHash32 returns a new Hash32 from a byte slice.  An error is returned if
// the number of bytes passed in is not len(Hash32{}).
func NewHash32(newHash []byte) (*Hash32, error) {
	var sh Hash32
	err := sh.SetBytes(newHash)
	if err != nil {
		return nil, err
	}
	return &sh, err
}

// NewHash32FromStr creates a Hash32 from a hash string.  The string should be
// the hexadecimal string of a byte-reversed hash, but any missing characters
// result in zero padding at the end of the Hash32.
func NewHash32FromStr(hash string) (*Hash32, error) {
	// Return error if hash string is too long.
	if len(hash) > MaxHashStringSize {
		return nil, ErrHashStrSize
	}

	// Hex decoder expects the hash to be a multiple of two.
	if len(hash)%2 != 0 {
		hash = "0" + hash
	}

	// Convert string hash to bytes.
	buf, err := hex.DecodeString(hash)
	if err != nil {
		return nil, err
	}

	// Un-reverse the decoded bytes, copying into in leading bytes of a
	// Hash32.  There is no need to explicitly pad the result as any
	// missing (when len(buf) < len(Hash32{})) bytes from the decoded hex string
	// will remain zeros at the end of the Hash32.
	var ret Hash32
	blen := len(buf)
	mid := blen / 2
	if blen%2 != 0 {
		mid++
	}
	blen--
	for i, b := range buf[:mid] {
		ret[i], ret[blen-i] = buf[blen-i], b
	}
	return &ret, nil
}
