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
	parsed, err := btcec.ParseDERSignature(encoded, btcec.S256())
	if err == nil {
		*s = (Signature)(*parsed)
	}
	return err
}
