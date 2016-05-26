package orderbook

import (
	"chain/cos/bc"
	"chain/database/pg"
	"chain/errors"

	"golang.org/x/net/context"
)

// Find* errors
var (
	ErrDuplicateOutpoints = errors.New("multiple orders found with duplicate outpoints") // should be impossible
	ErrNoAssets           = errors.New("no asset ids specified in find-orders request")
)

// FindOpenOrders finds open orders offering one of the given
// offeredAssetIDs and accepting one or more of the given
// paymentAssetIDs.  With zero paymentAssetIDs, returns all open
// orders offering any offeredAssetID.  With zero offeredAssetIDs,
// returns all open orders accepting any paymentAssetID.
func FindOpenOrders(ctx context.Context, offeredAssetIDs []bc.AssetID, paymentAssetIDs []bc.AssetID) ([]*OpenOrder, error) {
	var extra string
	var extraParams []interface{}

	if len(offeredAssetIDs) == 0 {
		if len(paymentAssetIDs) == 0 {
			return nil, ErrNoAssets
		}
		// Find all open orders that can be paid for with any of the paymentAssetIDs
		extra = `p.asset_id IN (SELECT unnest($1::text[]))`
		extraParams = append(extraParams, makeAssetIDStrs(paymentAssetIDs))
	} else {
		extra = `o.asset_id IN (SELECT unnest($1::text[]))`
		extraParams = append(extraParams, makeAssetIDStrs(offeredAssetIDs))
		if len(paymentAssetIDs) > 0 {
			extra += ` AND p.asset_id IN (SELECT unnest($2::text[]))`
			extraParams = append(extraParams, makeAssetIDStrs(paymentAssetIDs))
		}
	}
	return findOpenOrdersHelper(ctx, makeQuery(extra), extraParams...)
}

// FindOpenOrdersBySeller find open orders from the given seller account.
func FindOpenOrdersBySeller(ctx context.Context, accountID string) ([]*OpenOrder, error) {
	return findOpenOrdersHelper(ctx, makeQuery(`o.seller_id = $1`), accountID)
}

// FindOpenOrdersBySellerAndAsset find open orders from the given
// seller account and offering any of the given assetIDs.
func FindOpenOrdersBySellerAndAsset(ctx context.Context, accountID string, assetIDs []bc.AssetID) ([]*OpenOrder, error) {
	if len(assetIDs) == 0 {
		return FindOpenOrdersBySeller(ctx, accountID)
	}
	return findOpenOrdersHelper(ctx, makeQuery(`o.seller_id = $1 AND o.asset_id IN (SELECT unnest($2::text[]))`), accountID, makeAssetIDStrs(assetIDs))
}

// FindOpenOrderByOutpoint finds the open order with the given
// outpoint, if one exists.  Returns nil (no error) if it doesn't.
func FindOpenOrderByOutpoint(ctx context.Context, outpoint *bc.Outpoint) (*OpenOrder, error) {
	openOrders, err := findOpenOrdersHelper(ctx, makeQuery(`o.tx_hash = $1 AND o.index = $2`), outpoint.Hash, outpoint.Index)
	if err != nil {
		return nil, err
	}
	var result *OpenOrder
	for _, openOrder := range openOrders {
		if result == nil {
			result = openOrder
		} else {
			return nil, ErrDuplicateOutpoints // should be impossible
		}
	}
	return result, nil // note, result may be nil
}

func findOpenOrdersHelper(ctx context.Context, q string, args ...interface{}) ([]*OpenOrder, error) {
	rows, err := pg.Query(ctx, q, args...)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()

	var result []*OpenOrder

	var (
		lastTxHashStr string
		lastIndex     uint32
		openOrder     *OpenOrder
	)

	for rows.Next() {
		var (
			txHash                     bc.Hash
			index                      uint32
			offeredAssetID             bc.AssetID
			amount                     uint64
			script                     []byte
			sellerAccountID            string
			paymentAssetID             bc.AssetID
			offerAmount, paymentAmount uint64
		)

		err = rows.Scan(&txHash, &index, &offeredAssetID, &amount, &script, &sellerAccountID, &paymentAssetID, &offerAmount, &paymentAmount)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}

		if txHash.String() != lastTxHashStr || index != lastIndex {
			// wrap up the previous OpenOrder and start a new one
			if openOrder != nil {
				result = append(result, openOrder)
			}
			openOrder = &OpenOrder{
				Outpoint: bc.Outpoint{
					Hash:  txHash,
					Index: index,
				},
				OrderInfo: OrderInfo{
					SellerAccountID: sellerAccountID,
				},
				AssetAmount: bc.AssetAmount{AssetID: offeredAssetID, Amount: amount},
				Script:      script,
			}

			// Make the seller script directly visible in API responses.
			openOrder.OrderInfo.SellerScript, err = openOrder.SellerScript()
			if err != nil {
				return nil, errors.Wrap(err, "generate seller script")
			}
		}
		orderbookPrice := &Price{
			AssetID:       paymentAssetID,
			OfferAmount:   offerAmount,
			PaymentAmount: paymentAmount,
		}
		openOrder.Prices = append(openOrder.Prices, orderbookPrice)
		lastTxHashStr = txHash.String()
		lastIndex = index
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end scan")
	}

	if openOrder != nil {
		result = append(result, openOrder)
	}

	return result, nil
}

func makeAssetIDStrs(assetIDs []bc.AssetID) pg.Strings {
	result := make([]string, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		result = append(result, assetID.String())
	}
	return pg.Strings(result)
}

func makeQuery(extra string) string {
	const baseQuery = `
		SELECT o.tx_hash, o.index, o.asset_id, o.amount, o.script, o.seller_id, p.asset_id, p.offer_amount, p.payment_amount
		    FROM orderbook_utxos o, orderbook_prices p
		    WHERE o.tx_hash = p.tx_hash
		        AND o.index = p.index
		        AND 
	`
	return baseQuery + extra + ` ORDER BY o.tx_hash, o.index`
}
