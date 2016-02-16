package orderbook

import (
	"fmt"

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

	isOrderbook, sellerScript, prices, err := testOrderbookScript(output.Script)
	if err != nil {
		return errors.Wrap(err, "parsing utxo script")
	}
	if !isOrderbook {
		scriptStr, _ := txscript.DisasmString(output.Script)
		return fmt.Errorf("addOrderbookUTXO called on non-orderbook utxo [%s]", scriptStr)
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

	const q2 = `INSERT INTO orderbook_prices (tx_hash, index, asset_id, offer_amount, payment_amount) VALUES ($1, $2, $3, $4, $5)`
	for _, price := range prices {
		_, err := pg.FromContext(ctx).Exec(ctx, q2, hash, index, price.AssetID, price.OfferAmount, price.PaymentAmount)
		if err != nil {
			return errors.Wrap(err, "insert into orderbook_prices")
		}
	}

	return errors.Wrap(db.Commit(ctx), "commiting database tx")
}
