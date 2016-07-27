package appdb

import "chain/crypto/ed25519/hd25519"

func xpubsToStrings(keys []*hd25519.XPub) []string {
	var strings []string
	for _, k := range keys {
		strings = append(strings, k.String())
	}
	return strings
}

func xprvsToStrings(keys []*hd25519.XPrv) []string {
	var strings []string
	for _, k := range keys {
		strings = append(strings, k.String())
	}
	return strings
}

func stringsToXPubs(strings []string) ([]*hd25519.XPub, error) {
	res := make([]*hd25519.XPub, 0, len(strings))
	for _, s := range strings {
		xpub, err := hd25519.XPubFromString(s)
		if err != nil {
			return nil, err
		}
		res = append(res, xpub)
	}
	return res, nil
}

func stringsToXPrvs(strings []string) ([]*hd25519.XPrv, error) {
	res := make([]*hd25519.XPrv, 0, len(strings))
	for _, s := range strings {
		xprv, err := hd25519.XPrvFromString(s)
		if err != nil {
			return nil, err
		}
		res = append(res, xprv)
	}
	return res, nil
}
