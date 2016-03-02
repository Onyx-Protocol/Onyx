package asset

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txdb"
	"chain/fedchain"
	"chain/fedchain/hdkey"
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

func LoadAccountInfo(ctx context.Context, outs []*txdb.Output) ([]*txdb.Output, error) {
	return loadAccountInfo(ctx, outs)
}
