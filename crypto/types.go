package crypto

import (
	"encoding/json"

	"github.com/btcsuite/btcd/btcec"
)

type Signature btcec.Signature

func (s *Signature) MarshalJSON() ([]byte, error) {
	serialized := (*btcec.Signature)(s).Serialize()
	return json.Marshal(serialized)
}

func (s *Signature) UnmarshalJSON(encoded []byte) error {
	var b []byte
	err := json.Unmarshal(encoded, &b)
	if err != nil {
		return err
	}
	parsed, err := btcec.ParseDERSignature(b, btcec.S256())
	if err == nil {
		*s = (Signature)(*parsed)
	}
	return err
}
