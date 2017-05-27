package chainkd

import (
	"encoding/hex"
	"errors"

	"chain/crypto/ed25519/ecmath"
)

var (
	ErrBadKeyLen = errors.New("bad key length")
	ErrBadKeyStr = errors.New("bad key string")
)

func XPrvFromBytes(data []byte) (res XPrv, err error) {
	err = res.SetBytes(data)
	return
}

func XPubFromBytes(data []byte) (res XPub, err error) {
	err = res.SetBytes(data)
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

func (xpub *XPub) SetBytes(data []byte) error {
	if l := len(data); l != XPubSize {
		return ErrBadKeyLen
	}
	var (
		pubkey [32]byte
		P      ecmath.Point
	)
	copy(pubkey[:], data[:32])
	_, ok := P.Decode(pubkey)
	if !ok {
		return ErrBadKeyStr
	}
	copy(xpub.data[:], data[:])
	return nil
}

func (xprv XPrv) MarshalText() ([]byte, error) {
	hexBytes := make([]byte, hex.EncodedLen(len(xprv.Bytes())))
	hex.Encode(hexBytes, xprv.Bytes())
	return hexBytes, nil
}

func (xprv XPrv) Bytes() []byte {
	return xprv.data[:]
}

func (xprv *XPrv) SetBytes(data []byte) error {
	if l := len(data); l != XPrvSize {
		return ErrBadKeyLen
	}
	copy(xprv.data[:], data[:])
	return nil
}

func (xpub *XPub) UnmarshalText(inp []byte) error {
	if len(inp) != 2*XPubSize {
		return ErrBadKeyLen
	}
	var buf [64]byte
	_, err := hex.Decode(buf[:], inp)
	err = xpub.SetBytes(buf[:])
	if err != nil {
		return err
	}
	_, err = hex.Decode(xpub.data[:], inp)
	return err
}

func (xpub XPub) String() string {
	return hex.EncodeToString(xpub.Bytes())
}

func (xprv *XPrv) UnmarshalText(inp []byte) error {
	if len(inp) != 2*XPrvSize {
		return ErrBadKeyLen
	}
	_, err := hex.Decode(xprv.data[:], inp)
	return err
}

func (xprv XPrv) String() string {
	return hex.EncodeToString(xprv.Bytes())
}
