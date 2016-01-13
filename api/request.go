package api

import (
	"errors"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/api/smartcontracts/orderbook"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/fedchain/bc"
	"chain/net/http/httpjson"
)

// Data types and functions for marshaling/unmarshaling API requests

var (
	ErrNullAsset              = errors.New("asset type unspecified")
	ErrUnknownReceiverType    = errors.New("unknown request type")
	ErrUnknownSourceType      = errors.New("unknown source type")
	ErrUnknownDestinationType = errors.New("unknown destination type")
)

type Source struct {
	AssetID        *bc.AssetID `json:"asset_id"`
	Amount         uint64
	PaymentAssetID *bc.AssetID `json:"payment_asset_id"`
	PaymentAmount  uint64      `json:"payment_amount"`
	AccountID      string      `json:"account_id"`
	TxHash         *bc.Hash    `json:"transaction_hash"`
	Index          *uint32     `json:"index"`
	Type           string
}

func (source *Source) parse(ctx context.Context) (*asset.Source, error) {
	if source.Type == "account" || source.Type == "" {
		if source.AssetID == nil {
			return nil, httpjson.ErrBadRequest
		}
		assetAmount := &bc.AssetAmount{
			AssetID: *source.AssetID,
			Amount:  source.Amount,
		}
		return asset.NewAccountSource(ctx, assetAmount, source.AccountID), nil
	}
	if source.Type == "orderbook-redeem" {
		if source.PaymentAssetID == nil || source.TxHash == nil || source.Index == nil {
			return nil, httpjson.ErrBadRequest
		}
		outpoint := &bc.Outpoint{
			Hash:  *source.TxHash,
			Index: *source.Index,
		}
		openOrder, err := orderbook.FindOpenOrderByOutpoint(ctx, outpoint)
		if err != nil {
			return nil, err
		}
		if openOrder == nil {
			return nil, pg.ErrUserInputNotFound
		}
		paymentAmount := &bc.AssetAmount{
			AssetID: *source.PaymentAssetID,
			Amount:  source.PaymentAmount,
		}
		return orderbook.NewRedeemSource(openOrder, source.Amount, paymentAmount), nil
	}
	if source.Type == "orderbook-cancel" {
		if source.TxHash == nil || source.Index == nil {
			return nil, httpjson.ErrBadRequest
		}
		outpoint := &bc.Outpoint{
			Hash:  *source.TxHash,
			Index: *source.Index,
		}
		openOrder, err := orderbook.FindOpenOrderByOutpoint(ctx, outpoint)
		if err != nil {
			return nil, err
		}
		if openOrder == nil {
			return nil, pg.ErrUserInputNotFound
		}
		return orderbook.NewCancelSource(openOrder), nil
	}
	return nil, ErrUnknownSourceType
}

type Destination struct {
	AssetID         *bc.AssetID `json:"asset_id"`
	Amount          uint64
	AccountID       string             `json:"account_id,omitempty"`
	Address         chainjson.HexBytes `json:"address,omitempty"`
	IsChange        bool               `json:"is_change"`
	Metadata        chainjson.HexBytes `json:"metadata,omitempty"`
	OrderbookPrices []*orderbook.Price `json:"orderbook_prices,omitempty"`
	Script          chainjson.HexBytes `json:"script,omitempty"`
	Type            string
}

func (dest Destination) parse(ctx context.Context) (*asset.Destination, error) {
	if dest.AssetID == nil {
		return nil, ErrNullAsset
	}

	// backwards compatibility fix
	if dest.Type == "" && len(dest.Address) != 0 {
		dest.Type = "address"
	}

	assetAmount := &bc.AssetAmount{
		AssetID: *dest.AssetID,
		Amount:  dest.Amount,
	}

	switch dest.Type {
	case "account", "":
		return asset.NewAccountDestination(ctx, assetAmount, dest.AccountID, dest.IsChange, dest.Metadata)
	case "address":
		return asset.NewScriptDestination(ctx, assetAmount, dest.Address, dest.IsChange, dest.Metadata)
	case "orderbook":
		orderInfo := &orderbook.OrderInfo{
			SellerAccountID: dest.AccountID,
			Prices:          dest.OrderbookPrices,
		}
		return orderbook.NewDestinationWithScript(ctx, assetAmount, orderInfo, dest.IsChange, dest.Metadata, dest.Script)
	}
	return nil, ErrUnknownDestinationType
}

type Receiver struct {
	AccountID     string   `json:"account_id"`
	AddrIndex     []uint32 `json:"address_index"`
	IsChange      bool     `json:"is_change"`
	ManagerNodeID string   `json:"manager_node_id"`
	Script        chainjson.HexBytes
	OrderInfo     *orderbook.OrderInfo `json:"orderbook_info"`
	Type          string
}

func (receiver *Receiver) parse() (asset.Receiver, error) {
	// backwards compatibility fix
	if receiver.Type == "" && receiver.AccountID != "" {
		receiver.Type = "account"
	}

	switch receiver.Type {
	case "script", "":
		return asset.NewScriptReceiver(receiver.Script, receiver.IsChange), nil
	case "account":
		addr := &appdb.Address{
			AccountID:     receiver.AccountID,
			Index:         receiver.AddrIndex,
			ManagerNodeID: receiver.ManagerNodeID,
		}
		return asset.NewAccountReceiver(addr), nil
	case "orderbook":
		if receiver.OrderInfo == nil {
			return nil, httpjson.ErrBadRequest
		}
		return orderbook.NewReceiver(receiver.OrderInfo, receiver.IsChange, receiver.Script), nil
	}
	return nil, ErrUnknownReceiverType
}

type Template struct {
	Unsigned   *bc.TxData `json:"unsigned_hex"`
	BlockChain string     `json:"block_chain"`
	Inputs     []*asset.Input
	OutRecvs   []Receiver `json:"output_receivers"`
}

func (tpl *Template) parse(ctx context.Context) (*asset.TxTemplate, error) {
	result := &asset.TxTemplate{
		Unsigned:   tpl.Unsigned,
		BlockChain: tpl.BlockChain,
		Inputs:     tpl.Inputs,
	}
	for _, receiver := range tpl.OutRecvs {
		parsed, err := receiver.parse()
		if err != nil {
			return nil, err
		}
		result.OutRecvs = append(result.OutRecvs, parsed)
	}
	return result, nil
}

type BuildRequest struct {
	PrevTx   *Template          `json:"previous_transaction"`
	Sources  []*Source          `json:"inputs"`
	Dests    []*Destination     `json:"outputs"`
	Metadata chainjson.HexBytes `json:"metadata"`
	ResTime  time.Duration      `json:"reservation_duration"`
}

func (req *BuildRequest) parse(ctx context.Context) (*asset.TxTemplate, []*asset.Source, []*asset.Destination, error) {
	var (
		prevTx *asset.TxTemplate
		err    error
	)
	if req.PrevTx != nil {
		prevTx, err = req.PrevTx.parse(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	sources := make([]*asset.Source, 0, len(req.Sources))
	destinations := make([]*asset.Destination, 0, len(req.Dests))

	for _, source := range req.Sources {
		parsed, err := source.parse(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		sources = append(sources, parsed)
	}
	for _, destination := range req.Dests {
		parsed, err := destination.parse(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		destinations = append(destinations, parsed)
	}
	return prevTx, sources, destinations, nil
}
