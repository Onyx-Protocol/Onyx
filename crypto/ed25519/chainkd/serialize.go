package chainkd

import (
	"encoding/hex"
	"errors"
	"strconv"
)

var (
	ErrBadKeyLen = errors.New("bad key length")
	ErrBadKeyStr = errors.New("bad key string")
)

func XPrvFromBytes(data []byte) (res XPrv) {
	res.SetBytes(data)
	return
}

func XPubFromBytes(data []byte) (res XPub) {
	res.SetBytes(data)
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

func (xpub *XPub) SetBytes(data []byte) {
	if l := len(data); l != XPubSize {
		panic("chainkd: bad xpub length: " + strconv.Itoa(l))
	}
	copy(xpub.data[:], data[:])
}

func (xprv XPrv) MarshalText() ([]byte, error) {
	hexBytes := make([]byte, hex.EncodedLen(len(xprv.Bytes())))
	hex.Encode(hexBytes, xprv.Bytes())
	return hexBytes, nil
}

func (xprv XPrv) Bytes() []byte {
	return xprv.data[:]
}

func (xprv *XPrv) SetBytes(data []byte) {
	if l := len(data); l != XPrvSize {
		panic("chainkd: bad xprv length: " + strconv.Itoa(l))
	}
	copy(xprv.data[:], data[:])
}

func (xpub *XPub) UnmarshalText(inp []byte) error {
	if len(inp) != 2*XPubSize {
		return ErrBadKeyStr
	}
	_, err := hex.Decode(xpub.data[:], inp)
	return err
}

func (xpub XPub) String() string {
	return hex.EncodeToString(xpub.Bytes())
}

func (xprv *XPrv) UnmarshalText(inp []byte) error {
	if len(inp) != 2*XPrvSize {
		return ErrBadKeyStr
	}
	_, err := hex.Decode(xprv.data[:], inp)
	return err
}

func (xprv XPrv) String() string {
	return hex.EncodeToString(xprv.Bytes())
}
