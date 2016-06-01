package asset

import (
	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/cos"
	"chain/cos/bc"
)

func (ar *AccountReceiver) Addr() *appdb.Address {
	return ar.addr
}

func FC() *cos.FC {
	return fc
}

var BreakupChange = breakupChange

func AddBlock(ctx context.Context, b *bc.Block, conflicts []*bc.Tx) {
	addBlock(ctx, b, conflicts)
}
