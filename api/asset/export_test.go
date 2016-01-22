package asset

import (
	"chain/api/appdb"
	"chain/fedchain"
	"chain/fedchain-sandbox/hdkey"
)

func (ar *AccountReceiver) Addr() *appdb.Address {
	return ar.addr
}

func NewKey() (pub, priv *hdkey.XKey, err error) {
	return newKey()
}

func FC() *fedchain.FC {
	return fc
}
