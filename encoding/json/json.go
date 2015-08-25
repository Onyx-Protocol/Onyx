package json

import "encoding/hex"

type HexBytes []byte

func (h HexBytes) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h)), nil
}

func (h *HexBytes) UnmarshalText(text []byte) error {
	n := hex.DecodedLen(len(text))
	*h = make([]byte, n)
	_, err := hex.Decode(*h, text)
	return err
}
