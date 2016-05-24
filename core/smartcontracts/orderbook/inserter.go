package orderbook

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/database/pg"
	"chain/errors"
)

func addOrderbookUTXO(ctx context.Context, tx *bc.Tx, index int, sellerScript []byte, prices []*Price) error {
	db, ctx, err := pg.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "opening database tx")
	}
	defer db.Rollback(ctx)

	// TODO(bobg): batch these inserts
	out := tx.Outputs[index]
	const q1 = `
		INSERT INTO orderbook_utxos (tx_hash, index, seller_id, asset_id, amount, script)
		SELECT $1, $2, (SELECT account_id FROM addresses WHERE pk_script=$3), $4, $5, $6
	`
	_, err = pg.Exec(ctx, q1, tx.Hash, index, sellerScript, out.AssetID, out.Amount, out.Script)
	if err != nil {
		return errors.Wrap(err, "inserting into orderbook_utxos")
	}

	const q2 = `INSERT INTO orderbook_prices (tx_hash, index, asset_id, offer_amount, payment_amount) VALUES ($1, $2, $3, $4, $5)`
	for _, price := range prices {
		_, err := pg.Exec(ctx, q2, tx.Hash, index, price.AssetID, price.OfferAmount, price.PaymentAmount)
		if err != nil {
			return errors.Wrap(err, "insert into orderbook_prices")
		}
	}

	return errors.Wrap(db.Commit(ctx), "commiting database tx")
}
