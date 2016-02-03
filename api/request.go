package api

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/smartcontracts/orderbook"
	"chain/api/txbuilder"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/fedchain/bc"
)

// Data types and functions for marshaling/unmarshaling API requests

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

func (source *Source) parse(ctx context.Context) (*txbuilder.Source, error) {
	if source.Type == "account" || source.Type == "" {
		if source.AssetID == nil {
			return nil, errors.WithDetail(ErrBadBuildRequest, "asset type unspecified")
		}
		if source.Amount == 0 {
			return nil, errors.WithDetailf(ErrBadBuildRequest,
				"input for asset %s has zero amount", source.AssetID)
		}
		assetAmount := &bc.AssetAmount{
			AssetID: *source.AssetID,
			Amount:  source.Amount,
		}
		return asset.NewAccountSource(ctx, assetAmount, source.AccountID), nil
	}
	if source.Type == "orderbook-redeem" {
		if source.PaymentAssetID == nil {
			return nil, errors.WithDetail(ErrBadBuildRequest, "asset type unspecified")
		}
		if source.PaymentAmount == 0 {
			return nil, errors.WithDetailf(ErrBadBuildRequest,
				"input for asset %s has zero amount", *source.PaymentAssetID)
		}
		if source.TxHash == nil || source.Index == nil {
			return nil, errors.WithDetailf(ErrBadBuildRequest, "bad order")
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
			return nil, errors.WithDetailf(ErrBadBuildRequest, "bad order")
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
	return nil, errors.WithDetailf(ErrBadBuildRequest, "unknown source type `%s`", source.Type)
}

type Destination struct {
	AssetID         *bc.AssetID `json:"asset_id"`
	Amount          uint64
	AccountID       string             `json:"account_id,omitempty"`
	Address         chainjson.HexBytes `json:"address,omitempty"`
	Metadata        chainjson.HexBytes `json:"metadata,omitempty"`
	OrderbookPrices []*orderbook.Price `json:"orderbook_prices,omitempty"`
	Script          chainjson.HexBytes `json:"script,omitempty"`
	Type            string
}

func (dest Destination) parse(ctx context.Context) (*txbuilder.Destination, error) {
	if dest.AssetID == nil {
		return nil, errors.WithDetail(ErrBadBuildRequest, "asset type unspecified")
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
		return asset.NewAccountDestination(ctx, assetAmount, dest.AccountID, dest.Metadata)
	case "address":
		return asset.NewScriptDestination(ctx, assetAmount, dest.Address, dest.Metadata)
	case "orderbook":
		orderInfo := &orderbook.OrderInfo{
			SellerAccountID: dest.AccountID,
			Prices:          dest.OrderbookPrices,
		}
		return orderbook.NewDestinationWithScript(ctx, assetAmount, orderInfo, dest.Metadata, dest.Script)
	}
	return nil, errors.WithDetailf(ErrBadBuildRequest, "unknown destination type `%s`", dest.Type)
}

type Template struct {
	Unsigned   *bc.TxData `json:"unsigned_hex"`
	BlockChain string     `json:"block_chain"`
	Inputs     []*txbuilder.Input
}

func (tpl *Template) parse(ctx context.Context) (*txbuilder.Template, error) {
	result := &txbuilder.Template{
		Unsigned:   tpl.Unsigned,
		BlockChain: tpl.BlockChain,
		Inputs:     tpl.Inputs,
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

func (req *BuildRequest) parse(ctx context.Context) (*txbuilder.Template, []*txbuilder.Source, []*txbuilder.Destination, error) {
	var (
		prevTx *txbuilder.Template
		err    error
	)
	if req.PrevTx != nil {
		prevTx, err = req.PrevTx.parse(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	sources := make([]*txbuilder.Source, 0, len(req.Sources))
	destinations := make([]*txbuilder.Destination, 0, len(req.Dests))

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
