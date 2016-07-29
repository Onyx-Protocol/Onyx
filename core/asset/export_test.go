package asset

import (
	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/cos/state"
)

func (ar *AccountReceiver) AccountID() string {
	return ar.accountID
}

func Output(out state.Output, accountID string, keyIndex []uint32) *output {
	ret := output{
		Output:    out,
		AccountID: accountID,
	}

	copy(ret.keyIndex[:], keyIndex)
	return &ret
}

func FC() *cos.FC {
	return fc
}

var BreakupChange = breakupChange

func AddBlock(ctx context.Context, b *bc.Block) {
	indexAccountUTXOs(ctx, b)
}
