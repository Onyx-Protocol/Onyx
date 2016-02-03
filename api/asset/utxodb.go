package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/txdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/metrics"
)

type sqlUTXODB struct{}

// All UTXOs in the system.
var utxoDB = utxodb.New(sqlUTXODB{})

func (sqlUTXODB) LoadUTXOs(ctx context.Context, accountID string, assetID bc.AssetID) (resvOuts []*utxodb.UTXO, err error) {
	bcOuts, err := txdb.LoadUTXOs(ctx, accountID, assetID)
	if err != nil {
		return nil, errors.Wrap(err, "load blockchain outputs")
	}
	return bcOuts, nil
}

func isSpent(ctx context.Context, p bc.Outpoint, v state.ViewReader) bool {
	o := v.Output(ctx, p)
	return o != nil && o.Spent
}

func (sqlUTXODB) SaveReservations(ctx context.Context, utxos []*utxodb.UTXO, exp time.Time) error {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		UPDATE account_utxos
		SET reserved_until=$3
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	var txHashes []string
	var indexes []uint32
	for _, u := range utxos {
		txHashes = append(txHashes, u.Outpoint.Hash.String())
		indexes = append(indexes, u.Outpoint.Index)
	}
	_, err := pg.FromContext(ctx).Exec(ctx, q, pg.Strings(txHashes), pg.Uint32s(indexes), exp)
	return errors.Wrap(err, "update utxo reserve expiration")
}
