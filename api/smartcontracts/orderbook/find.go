package orderbook

import (
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"

	"golang.org/x/net/context"
)

// Find* errors
var ErrDuplicateOutpoints = errors.New("multiple orders found with duplicate outpoints") // should be impossible

// FindOpenOrders find open orders offering the given offeredAssetID
// and accepting one or more of the given paymentAssetIDs.  With zero
// paymentAssetIDs, returns all open orders offering offeredAssetID.
func FindOpenOrders(ctx context.Context, offeredAssetID bc.AssetID, paymentAssetIDs []bc.AssetID) (<-chan *OpenOrder, error) {
	extra := `u.asset_id = $1`
	offeredAssetIDStr := offeredAssetID.String()
	if len(paymentAssetIDs) == 0 {
		return findOpenOrdersHelper(ctx, makeQuery(extra), offeredAssetIDStr)
	}
	extra += ` AND p.asset_id IN (SELECT unnest($2::text[]))`
	return findOpenOrdersHelper(ctx, makeQuery(extra), offeredAssetIDStr, makeAssetIDStrs(paymentAssetIDs))
}

// FindOpenOrdersBySeller find open orders from the given seller account.
func FindOpenOrdersBySeller(ctx context.Context, accountID string) (<-chan *OpenOrder, error) {
	return findOpenOrdersHelper(ctx, makeQuery(`ou.seller_id = $1`), accountID)
}

// FindOpenOrdersBySellerAndAsset find open orders from the given
// seller account and offering any of the given assetIDs.
func FindOpenOrdersBySellerAndAsset(ctx context.Context, accountID string, assetIDs []bc.AssetID) (<-chan *OpenOrder, error) {
	if len(assetIDs) == 0 {
		return FindOpenOrdersBySeller(ctx, accountID)
	}
	return findOpenOrdersHelper(ctx, makeQuery(`ou.seller_id = $1 AND u.asset_id IN (SELECT unnest($2::text[]))`), accountID, makeAssetIDStrs(assetIDs))
}

// FindOpenOrderByOutpoint finds the open order with the given
// outpoint, if one exists.  Returns nil (no error) if it doesn't.
func FindOpenOrderByOutpoint(ctx context.Context, outpoint *bc.Outpoint) (*OpenOrder, error) {
	ch, err := findOpenOrdersHelper(ctx, makeQuery(`u.tx_hash = $1 AND u.index = $2`), outpoint.Hash, outpoint.Index)
	if err != nil {
		return nil, err
	}
	var result *OpenOrder
	for order := range ch {
		if result == nil {
			result = order
		} else {
			return nil, ErrDuplicateOutpoints // should be impossible
		}
	}
	return result, nil // note, result may be nil
}

func findOpenOrdersHelper(ctx context.Context, q string, args ...interface{}) (<-chan *OpenOrder, error) {
	rows, err := pg.FromContext(ctx).Query(ctx, q, args...)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}

	result := make(chan *OpenOrder)

	go func() {
		defer rows.Close()
		defer close(result)

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

			if txHash.String() != lastTxHashStr || index != lastIndex {
				// wrap up the previous OpenOrder and start a new one
				if openOrder != nil {
					result <- openOrder
				}
				openOrder = &OpenOrder{
					Outpoint: bc.Outpoint{
						Hash:  txHash,
						Index: index,
					},
					OrderInfo: OrderInfo{
						SellerAccountID: sellerAccountID,
					},
					AssetAmount: bc.AssetAmount{
						AssetID: offeredAssetID,
						Amount:  amount,
					},
					Script: script,
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
		if openOrder != nil {
			result <- openOrder
		}
	}()

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
		SELECT u.tx_hash, u.index, u.asset_id, u.amount, u.script, ou.seller_id, p.asset_id, p.offer_amount, p.payment_amount
		    FROM utxos u, orderbook_utxos ou, orderbook_prices p
		    WHERE u.tx_hash = ou.tx_hash
		        AND u.index = ou.index
		        AND u.tx_hash = p.tx_hash
		        AND u.index = p.index
		        AND NOT EXISTS (SELECT 1 FROM pool_inputs pi WHERE pi.tx_hash = u.tx_hash AND pi.index = u.index)
		        AND 
	`
	return baseQuery + extra + ` ORDER BY u.tx_hash, u.index`
}
