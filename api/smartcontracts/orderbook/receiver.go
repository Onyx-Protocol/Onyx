package orderbook

import (
	"encoding/json"

	"golang.org/x/net/context"

	"chain/api/txbuilder"
	chainjson "chain/encoding/json"
	"chain/fedchain/bc"
)

type orderbookReceiver struct {
	orderInfo *OrderInfo
	pkscript  []byte
}

func (receiver *orderbookReceiver) PKScript() []byte { return receiver.pkscript }
func (receiver *orderbookReceiver) AccumulateUTXO(ctx context.Context, outpoint *bc.Outpoint, txOutput *bc.TxOutput, inserters []txbuilder.UTXOInserter) ([]txbuilder.UTXOInserter, error) {
	// Find or create an item in utxoInserters that is an
	// orderbookUTXOInserter
	var orderbookInserter *orderbookUTXOInserter
	for _, inserter := range inserters {
		var ok bool
		if orderbookInserter, ok = inserter.(*orderbookUTXOInserter); ok {
			break
		}
	}
	if orderbookInserter == nil {
		orderbookInserter = &orderbookUTXOInserter{}
		inserters = append(inserters, orderbookInserter)
	}
	orderbookInserter.add(outpoint, txOutput, receiver)
	return inserters, nil
}
func (receiver *orderbookReceiver) MarshalJSON() ([]byte, error) {
	dict := make(map[string]interface{})
	dict["script"] = chainjson.HexBytes(receiver.pkscript)
	dict["orderbook_info"] = receiver.orderInfo
	dict["type"] = "orderbook"
	return json.Marshal(dict)
}

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
