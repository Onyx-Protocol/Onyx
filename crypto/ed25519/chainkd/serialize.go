package chainkd

import (
	"encoding/hex"
	"errors"
	"strconv"
)

const (
	extendedPublicKeySize  = 64
	extendedPrivateKeySize = 64
)

var (
	ErrBadKeyLen = errors.New("bad key length")
	ErrBadKeyStr = errors.New("bad key string")
)

func XPrvFromBytes(data []byte) (res XPrv) {
	if l := len(data); l != extendedPrivateKeySize {
		panic("chainkd: bad xprv length: " + strconv.Itoa(l))
	}
	copy(res.data[:], data[:])
	return
}

func XPubFromBytes(data []byte) (res XPub) {
	if l := len(data); l != extendedPublicKeySize {
		panic("chainkd: bad xpub length: " + strconv.Itoa(l))
	}
	copy(res.data[:], data[:])
	return
}

func (xpub XPub) MarshalText() ([]byte, error) {
	hexBytes := make([]byte, hex.EncodedLen(len(xpub.Bytes())))
	hex.Encode(hexBytes, xpub.Bytes())
	return hexBytes, nil
}

func (xpub XPub) Bytes() []byte {
	return xpub.data[:]
}

func (xprv XPrv) MarshalText() ([]byte, error) {
	hexBytes := make([]byte, hex.EncodedLen(len(xprv.Bytes())))
	hex.Encode(hexBytes, xprv.Bytes())
	return hexBytes, nil
}

func (xprv XPrv) Bytes() []byte {
	return xprv.data[:]
}

func (xpub *XPub) UnmarshalText(inp []byte) error {
	if len(inp) != 2*extendedPublicKeySize {
		return ErrBadKeyStr
	}
	_, err := hex.Decode(xpub.data[:], inp)
	return err
}

func (xpub XPub) String() string {
	return hex.EncodeToString(xpub.Bytes())
}

func (xprv *XPrv) UnmarshalText(inp []byte) error {
	if len(inp) != 2*extendedPrivateKeySize {
		return ErrBadKeyStr
	}
	_, err := hex.Decode(xprv.data[:], inp)
	return err
}

func (xprv XPrv) String() string {
	return hex.EncodeToString(xprv.Bytes())
}
