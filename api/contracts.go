package api

import (
	"golang.org/x/net/context"

	"chain/api/smartcontracts/orderbook"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/net/http/httpjson"
)

type globalFindOrder struct {
	OfferedAssetID  bc.AssetID   `json:"offered_asset_id"`
	PaymentAssetIDs []bc.AssetID `json:"payment_asset_ids"`
}

func findOrders(ctx context.Context, req globalFindOrder) ([]*orderbook.OpenOrder, error) {
	qvals := httpjson.Request(ctx).URL.Query()
	if status, ok := qvals["status"]; !ok || status[0] != "open" {
		// TODO(tessr): find closed orders
		return nil, errors.Wrap(httpjson.ErrBadRequest, "unimplemented: find all orders")
	}
	oc, err := orderbook.FindOpenOrders(ctx, req.OfferedAssetID, req.PaymentAssetIDs)
	if err != nil {
		return nil, errors.Wrap(err, "finding orders by offered and payment asset ids")
	}

	orders := make([]*orderbook.OpenOrder, 0)
	for order := range oc {
		orders = append(orders, order)
	}

	return orders, nil
}

func findAccountOrders(ctx context.Context, accountID string) ([]*orderbook.OpenOrder, error) {
	qvals := httpjson.Request(ctx).URL.Query()
	if status, ok := qvals["status"]; !ok || status[0] != "open" {
		// TODO(tessr): find closed orders
		return nil, errors.Wrap(httpjson.ErrBadRequest, "unimplemented: find all orders")
	}
	var (
		oc  <-chan *orderbook.OpenOrder
		err error
	)
	if aids, ok := qvals["asset_id"]; ok {
		var assetIDs []bc.AssetID
		for _, id := range aids {
			var assetID bc.AssetID
			err = assetID.UnmarshalText([]byte(id))
			if err != nil {
				return nil, errors.Wrap(httpjson.ErrBadRequest, "invalid assetID")
			}
			assetIDs = append(assetIDs, assetID)
		}
		oc, err = orderbook.FindOpenOrdersBySellerAndAsset(ctx, accountID, assetIDs)
		if err != nil {
			return nil, errors.Wrap(err, "finding orders by seller and asset")
		}
	} else {
		oc, err = orderbook.FindOpenOrdersBySeller(ctx, accountID)
		return nil, errors.Wrap(err, "finding orders by seller")
	}

	orders := make([]*orderbook.OpenOrder, 0)
	for order := range oc {
		orders = append(orders, order)
	}

	return orders, nil
}
