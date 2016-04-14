package orderbook

import (
	"golang.org/x/net/context"

	"chain/api/txbuilder"
	"chain/cos/bc"
)

type orderbookReceiver struct {
	orderInfo *OrderInfo
	pkscript  []byte
}

func (receiver *orderbookReceiver) PKScript() []byte { return receiver.pkscript }

func NewReceiver(orderInfo *OrderInfo, pkscript []byte) *orderbookReceiver {
	return &orderbookReceiver{
		orderInfo: orderInfo,
		pkscript:  pkscript,
	}
}

// NewDestination creates an txbuilder.Destination that commits assets to
// an Orderbook contract.
func NewDestination(ctx context.Context, assetAmount *bc.AssetAmount, orderInfo *OrderInfo, metadata []byte) (*txbuilder.Destination, error) {
	return NewDestinationWithScript(ctx, assetAmount, orderInfo, metadata, nil)
}

// NewDestinationWithScript creates an txbuilder.Destination that commits
// assets to an Orderbook contract, and short-circuits
// script/address-creation.
func NewDestinationWithScript(ctx context.Context, assetAmount *bc.AssetAmount, orderInfo *OrderInfo, metadata, pkscript []byte) (*txbuilder.Destination, error) {
	if pkscript == nil {
		var err error
		pkscript, _, _, err = orderInfo.generateScript(ctx, nil)
		if err != nil {
			return nil, err
		}
	}
	result := &txbuilder.Destination{
		AssetAmount: *assetAmount,
		Metadata:    metadata,
		Receiver:    NewReceiver(orderInfo, pkscript),
	}
	return result, nil
}
