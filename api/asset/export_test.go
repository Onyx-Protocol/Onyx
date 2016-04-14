package asset

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txdb"
	"chain/cos"
)

func (ar *AccountReceiver) Addr() *appdb.Address {
	return ar.addr
}

func FC() *cos.FC {
	return fc
}

func LoadAccountInfo(ctx context.Context, outs []*txdb.Output) ([]*txdb.Output, error) {
	return loadAccountInfo(ctx, outs)
}

var BreakupChange = breakupChange
