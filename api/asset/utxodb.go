package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/txscript"
	"chain/fedchain-sandbox/wire"
	"chain/log"
	"chain/metrics"
)

type sqlUTXODB struct{}

func (sqlUTXODB) LoadUTXOs(ctx context.Context, bucketID, assetID string) ([]*utxodb.UTXO, error) {
	log.Messagef(ctx, "loading full utxo set")
	t0 := time.Now()
	const q = `
		SELECT amount, reserved_until, address_id, txid, index
		FROM utxos
		WHERE bucket_id=$1 AND asset_id=$2
	`
	rows, err := pg.FromContext(ctx).Query(q, bucketID, assetID)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()
	var utxos []*utxodb.UTXO
	for rows.Next() {
		u := &utxodb.UTXO{
			BucketID: bucketID,
			AssetID:  assetID,
		}
		var txid string
		err = rows.Scan(
			&u.Amount,
			&u.ResvExpires,
			&u.AddressID,
			&txid,
			&u.Outpoint.Index,
		)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		h, err := wire.NewHash32FromStr(txid)
		if err != nil {
			return nil, errors.Wrap(err, "decode hash")
		}
		u.Outpoint.Hash = *h
		u.ResvExpires = u.ResvExpires.UTC()
		utxos = append(utxos, u)
		if len(utxos)%1e6 == 0 {
			log.Messagef(ctx, "loaded %d utxos so far", len(utxos))
		}
	}
	log.Messagef(ctx, "loaded %d utxos done (%v)", len(utxos), time.Since(t0))
	return utxos, errors.Wrap(rows.Err(), "rows")
}

func (sqlUTXODB) SaveReservations(ctx context.Context, utxos []*utxodb.UTXO, exp time.Time) error {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		UPDATE utxos
		SET reserved_until=$3
		WHERE (txid, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	var txids []string
	var indexes []uint32
	for _, u := range utxos {
		txids = append(txids, u.Outpoint.Hash.String())
		indexes = append(indexes, u.Outpoint.Index)
	}
	_, err := pg.FromContext(ctx).Exec(q, pg.Strings(txids), pg.Uint32s(indexes), exp)
	return errors.Wrap(err, "update utxo reserve expiration")
}

// ApplyTx updates the output set to reflect
// the effects of tx. It deletes consumed utxos
// and inserts newly-created outputs.
// Must be called inside a transaction.
func (sqlUTXODB) ApplyTx(ctx context.Context, tx *wire.MsgTx) (deleted, inserted []*utxodb.UTXO, err error) {
	defer metrics.RecordElapsed(time.Now())
	now := time.Now()
	hash := tx.TxSha()
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	inserted, err = insertUTXOs(ctx, hash, tx.TxOut)
	if err != nil {
		return nil, nil, errors.Wrap(err, "insert")
	}

	// Activity items rely on the utxo set, so they should be created after
	// the output utxos are created but before the input utxos are removed.
	err = appdb.CreateActivityItems(ctx, tx, now)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating activity items")
	}

	deleted, err = deleteUTXOs(ctx, tx.TxIn)
	if err != nil {
		return nil, nil, errors.Wrap(err, "delete")
	}
	return deleted, inserted, err
}

type outputSet struct {
	txid    string
	index   pg.Uint32s
	assetID pg.Strings
	amount  pg.Int64s
	addr    pg.Strings
}

func deleteUTXOs(ctx context.Context, txins []*wire.TxIn) ([]*utxodb.UTXO, error) {
	defer metrics.RecordElapsed(time.Now())
	var (
		txid  []string
		index []uint32
	)
	for _, in := range txins {
		txid = append(txid, in.PreviousOutPoint.Hash.String())
		index = append(index, in.PreviousOutPoint.Index)
	}

	const q = `
		WITH outpoints AS (
			SELECT unnest($1::text[]), unnest($2::int[])
		)
		DELETE FROM utxos
		WHERE (txid, index) IN (TABLE outpoints)
		RETURNING bucket_id, asset_id, address_id, txid, index
	`
	rows, err := pg.FromContext(ctx).Query(q, pg.Strings(txid), pg.Uint32s(index))
	if err != nil {
		return nil, errors.Wrap(err, "delete")
	}
	defer rows.Close()
	var deleted []*utxodb.UTXO
	for rows.Next() {
		u := new(utxodb.UTXO)
		var txid string
		err = rows.Scan(&u.BucketID, &u.AssetID, &u.AddressID, &txid, &u.Outpoint.Index)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		h, err := wire.NewHash32FromStr(txid)
		if err != nil {
			return nil, errors.Wrap(err, "decode hash")
		}
		u.Outpoint.Hash = *h
		deleted = append(deleted, u)
	}
	return deleted, rows.Err()
}

func insertUTXOs(ctx context.Context, hash wire.Hash32, txouts []*wire.TxOut) ([]*utxodb.UTXO, error) {
	defer metrics.RecordElapsed(time.Now())
	outs := &outputSet{txid: hash.String()}
	err := addTxOutputs(outs, txouts)
	if err != nil {
		return nil, err
	}

	const q = `
		WITH newouts AS (
			SELECT
				unnest($2::int[]) idx,
				unnest($3::text[]) asset_id,
				unnest($4::bigint[]) amount,
				unnest($5::text[]) addr
		),
		recouts AS (
			SELECT
				$1::text, idx, asset_id, newouts.amount, id, bucket_id, wallet_id
			FROM addresses
			INNER JOIN newouts ON address=addr
		)
		INSERT INTO utxos
			(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		TABLE recouts
		RETURNING bucket_id, asset_id, amount, address_id, txid, index
	`
	rows, err := pg.FromContext(ctx).Query(q,
		outs.txid,
		outs.index,
		outs.assetID,
		outs.amount,
		outs.addr,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var inserted []*utxodb.UTXO
	for rows.Next() {
		u := new(utxodb.UTXO)
		var txid string
		err = rows.Scan(&u.BucketID, &u.AssetID, &u.Amount, &u.AddressID, &txid, &u.Outpoint.Index)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		h, err := wire.NewHash32FromStr(txid)
		if err != nil {
			return nil, errors.Wrap(err, "decode hash")
		}
		u.Outpoint.Hash = *h
		inserted = append(inserted, u)
	}
	return inserted, rows.Err()
}

func addTxOutputs(outs *outputSet, txouts []*wire.TxOut) error {
	for i, txo := range txouts {
		outs.index = append(outs.index, uint32(i))
		outs.assetID = append(outs.assetID, txo.AssetID.String())
		outs.amount = append(outs.amount, txo.Value)

		addr, err := txscript.PkScriptAddr(txo.PkScript)
		if err != nil {
			return err
		}
		outs.addr = append(outs.addr, addr.String())
	}

	return nil
}
