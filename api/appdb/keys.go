package appdb

import "chain/fedchain-sandbox/hdkey"

func keysToXPubs(keys []*hdkey.XKey) []string {
	var xpubs []string
	for _, k := range keys {
		xpubs = append(xpubs, k.String())
	}
	return xpubs
}

func xpubsToKeys(xpubs []string) ([]*hdkey.XKey, error) {
	var keys []*hdkey.XKey
	for _, x := range xpubs {
		key, err := hdkey.NewXKey(x)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}
