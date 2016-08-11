package json

import (
	"encoding/hex"
	"encoding/json"
)

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

type Map []byte

func (m Map) MarshalJSON() ([]byte, error) {
	return m, nil
}

func (m *Map) UnmarshalJSON(text []byte) error {
	var check map[string]*json.RawMessage
	err := json.Unmarshal(text, &check)
	if err != nil {
		return err
	}
	*m = text
	return nil
}
