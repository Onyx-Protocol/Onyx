package appdb

import "chain/fedchain/hdkey"

func keysToStrings(keys []*hdkey.XKey) []string {
	var strings []string
	for _, k := range keys {
		strings = append(strings, k.String())
	}
	return strings
}

func stringsToKeys(strings []string) ([]*hdkey.XKey, error) {
	var keys []*hdkey.XKey
	for _, x := range strings {
		key, err := hdkey.NewXKey(x)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}
