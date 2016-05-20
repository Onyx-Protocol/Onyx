package asset

import (
	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/txdb"
	"chain/cos"
	"chain/cos/bc"
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

func AddBlock(ctx context.Context, b *bc.Block, conflicts []*bc.Tx) {
	addBlock(ctx, b, conflicts)
}
