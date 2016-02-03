package orderbook

import (
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
)

func addOrderbookUTXO(ctx context.Context, hash bc.Hash, index int, output *bc.TxOutput) error {
	db, ctx, err := pg.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "opening database tx")
	}
	defer db.Rollback(ctx)

	sellerScript, err := extractSellerScript(output.Script)
	if err != nil {
		return errors.Wrap(err, "extracting seller script")
	}

	// TODO(bobg): batch these inserts
	const q1 = `
		INSERT INTO orderbook_utxos (tx_hash, index, seller_id)
		SELECT $1, $2, (SELECT account_id FROM addresses WHERE pk_script=$3)
	`
	_, err = pg.FromContext(ctx).Exec(ctx, q1, hash, index, sellerScript)
	if err != nil {
		return errors.Wrap(err, "inserting into orderbook_utxos")
	}

	prices, err := extractPrices(output.Script)
	if err != nil {
		return errors.Wrap(err, "extracting prices")
	}

	const q2 = `INSERT INTO orderbook_prices (tx_hash, index, asset_id, offer_amount, payment_amount) VALUES ($1, $2, $3, $4, $5)`
	for _, price := range prices {
		_, err := pg.FromContext(ctx).Exec(ctx, q2, hash, index, price.AssetID, price.OfferAmount, price.PaymentAmount)
		if err != nil {
			return errors.Wrap(err, "insert into orderbook_prices")
		}
	}

	return errors.Wrap(db.Commit(ctx), "commiting database tx")
}

func extractPrices(pkScript []byte) ([]*Price, error) {
	data, err := txscript.PushedData(pkScript)
	if err != nil {
		return nil, errors.Wrap(err, "getting push data")
	}
	if len(data) < 6 || len(data)%3 != 0 {
		return nil, errors.Wrap(err, "incorrect number of script parameters")
	}

	data = data[1 : len(data)-2]

	var prices []*Price
	for i := 0; i < len(data); i += 3 {
		var price Price
		if len(data[i]) != len(bc.Hash{}) {
			return nil, errors.Wrap(err, "invalid asset id in parameters")
		}
		copy(price.AssetID[:], data[i])

		num, err := txscript.MakeScriptNumWithMaxLen(data[i+1], false, len(data[i+1]))
		if err != nil {
			return nil, errors.Wrap(err, "invalid payment amount in parameters")
		}
		price.PaymentAmount = uint64(num)

		num, err = txscript.MakeScriptNumWithMaxLen(data[i+2], false, len(data[i+2]))
		if err != nil {
			return nil, errors.Wrap(err, "invalid offer amount in parameters")
		}
		price.OfferAmount = uint64(num)

		prices = append(prices, &price)
	}

	return prices, nil
}
