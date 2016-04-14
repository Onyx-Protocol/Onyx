package txbuilder

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/txscript"
)

type retireReceiver struct{}

func (r retireReceiver) PKScript() []byte {
	return []byte{txscript.OP_RETURN}
}

// NewRetireDestination returns a Destination
// that will use the retire the quantity of the
// asset specified in the AssetAmount
func NewRetireDestination(ctx context.Context, assetAmount *bc.AssetAmount, metadata []byte) *Destination {
	dest := &Destination{
		AssetAmount: *assetAmount,
		Metadata:    metadata,
		Receiver:    retireReceiver{},
	}
	return dest
}
