package orderbook

import (
	"golang.org/x/net/context"

	"chain/api/txdb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type orderbookUTXOInserter struct {
	txdbOutputs []*txdb.Output
	receivers   []*orderbookReceiver
}

func (inserter *orderbookUTXOInserter) add(outpoint *bc.Outpoint, txOutput *bc.TxOutput, receiver *orderbookReceiver) {
	txdbOutput := &txdb.Output{
		Output: state.Output{
			TxOutput: *txOutput,
			Outpoint: *outpoint,
		},
	}
	inserter.txdbOutputs = append(inserter.txdbOutputs, txdbOutput)
	inserter.receivers = append(inserter.receivers, receiver)
}

func (inserter *orderbookUTXOInserter) InsertUTXOs(ctx context.Context) ([]*txdb.Output, error) {
	db := pg.FromContext(ctx)
	_ = db.(pg.Tx) // panics if not in a db transaction

	err := txdb.InsertPoolOutputs(ctx, inserter.txdbOutputs)
	if err != nil {
		return nil, err
	}
	for i, txdbOutput := range inserter.txdbOutputs {
		orderInfo := inserter.receivers[i].orderInfo
		outpoint := txdbOutput.Outpoint

		// TODO(bobg): batch these INSERTs
		const q1 = `
			INSERT INTO orderbook_utxos (tx_hash, index, seller_id)
			VALUES ($1, $2, $3)
		`
		_, err = db.Exec(ctx, q1, outpoint.Hash, outpoint.Index, orderInfo.SellerAccountID)

		const q2 = `INSERT INTO orderbook_prices (tx_hash, index, asset_id, offer_amount, payment_amount) VALUES ($1, $2, $3, $4, $5)`
		for _, price := range orderInfo.Prices {
			_, err := db.Exec(ctx, q2, outpoint.Hash, outpoint.Index, price.AssetID, price.OfferAmount, price.PaymentAmount)
			if err != nil {
				return nil, errors.Wrap(err, "insert into orderbook_prices")
			}
		}
	}
	return inserter.txdbOutputs, nil
}
