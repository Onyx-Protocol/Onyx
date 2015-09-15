package hdkey

import (
	"github.com/btcsuite/btcutil/hdkeychain"

	"chain/errors"
)

// XKey represents an extended key,
// with additional methods to marshal and unmarshal as text,
// for JSON encoding.
// The embedded type carries methods with it;
// see its documentation for details.
type XKey struct {
	hdkeychain.ExtendedKey
}

func (k XKey) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

func (k *XKey) UnmarshalText(p []byte) error {
	key, err := hdkeychain.NewKeyFromString(string(p))
	if err != nil {
		return errors.Wrap(err, "unmarshal XKey")
	}
	k.ExtendedKey = *key
	return nil
}
