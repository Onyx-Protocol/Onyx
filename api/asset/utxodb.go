package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/txscript"
	"chain/fedchain/bc"
	"chain/log"
	"chain/metrics"
)

type sqlUTXODB struct{}

func (sqlUTXODB) LoadUTXOs(ctx context.Context, bucketID, assetID string) ([]*utxodb.UTXO, error) {
	log.Messagef(ctx, "loading full utxo set")
	t0 := time.Now()
	const q = `
		SELECT amount, reserved_until, txid, index
		FROM utxos
		WHERE account_id=$1 AND asset_id=$2
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
			&txid,
			&u.Outpoint.Index,
		)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		h, err := bc.ParseHash(txid)
		if err != nil {
			return nil, errors.Wrap(err, "decode hash")
		}
		u.Outpoint.Hash = h
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
func (sqlUTXODB) ApplyTx(ctx context.Context, tx *bc.Tx, outRecs []*utxodb.Receiver) (deleted, inserted []*utxodb.UTXO, err error) {
	defer metrics.RecordElapsed(time.Now())
	now := time.Now()
	hash := tx.Hash()
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	insUTXOs, err := insertUTXOs(ctx, hash, tx.Outputs, outRecs)
	if err != nil {
		return nil, nil, errors.Wrap(err, "insert")
	}
	var localUTXOs []*appdb.UTXO
	for _, utxo := range insUTXOs {
		if utxo.WalletID != "" {
			localUTXOs = append(localUTXOs, utxo)
		}
	}

	// Activity items rely on the utxo set, so they should be created after
	// the output utxos are created but before the input utxos are removed.
	err = appdb.WriteActivity(ctx, tx, localUTXOs, now)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating activity items")
	}

	deleted, err = deleteUTXOs(ctx, tx.Inputs)
	if err != nil {
		return nil, nil, errors.Wrap(err, "delete")
	}
	for _, u := range localUTXOs {
		inserted = append(inserted, u.UTXO)
	}
	return deleted, inserted, err
}

// utxoSet holds a set of utxo record values
// to be inserted into the db.
type utxoSet struct {
	txid     string
	index    pg.Uint32s
	assetID  pg.Strings
	amount   pg.Int64s
	addr     pg.Strings
	bucketID pg.Strings
	walletID pg.Strings
	aIndex   pg.Int64s
}

func deleteUTXOs(ctx context.Context, txins []*bc.TxInput) ([]*utxodb.UTXO, error) {
	defer metrics.RecordElapsed(time.Now())
	var (
		txid  []string
		index []uint32
	)
	for _, in := range txins {
		txid = append(txid, in.Previous.Hash.String())
		index = append(index, in.Previous.Index)
	}

	const q = `
		WITH outpoints AS (
			SELECT unnest($1::text[]), unnest($2::bigint[])
		)
		DELETE FROM utxos
		WHERE (txid, index) IN (TABLE outpoints)
		RETURNING account_id, asset_id, txid, index
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
		err = rows.Scan(&u.BucketID, &u.AssetID, &txid, &u.Outpoint.Index)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		h, err := bc.ParseHash(txid)
		if err != nil {
			return nil, errors.Wrap(err, "decode hash")
		}
		u.Outpoint.Hash = h
		deleted = append(deleted, u)
	}
	return deleted, rows.Err()
}

func insertUTXOs(ctx context.Context, hash bc.Hash, txouts []*bc.TxOutput, recs []*utxodb.Receiver) ([]*appdb.UTXO, error) {
	if len(txouts) != len(recs) {
		return nil, errors.New("length mismatch")
	}
	defer metrics.RecordElapsed(time.Now())

	// This function inserts utxos into the db, and maps
	// them to receiver info (bucket id and addr index).
	// There are three cases:
	// 1. UTXO pays change or to an "immediate" bucket receiver.
	//    In this case, we get the receiver info from recs
	//    (which came from the client and was validated
	//    in FinalizeTx).
	// 2. UTXO pays to an address receiver record.
	//    In this case, we get the receiver info from
	//    the addresses table (and eventually delete
	//    the record).
	// 3. UTXO pays to an unknown address.
	//    In this case, there is no receiver info.
	insert, err := initAddrInfoFromRecs(hash, txouts, recs) // case 1
	if err != nil {
		return nil, err
	}
	err = loadAddrInfoFromDB(ctx, insert) // case 2
	if err != nil {
		return nil, err
	}

	outs := &utxoSet{txid: hash.String()}
	for i, u := range insert {
		outs.index = append(outs.index, uint32(i))
		outs.assetID = append(outs.assetID, u.AssetID)
		outs.amount = append(outs.amount, int64(u.Amount))
		outs.bucketID = append(outs.bucketID, u.BucketID)
		outs.walletID = append(outs.walletID, u.WalletID)
		outs.aIndex = append(outs.aIndex, toKeyIndex(u.AddrIndex[:]))
	}

	const q = `
		INSERT INTO utxos (
			txid, index, asset_id, amount,
			account_id, manager_node_id, addr_index
		)
		SELECT
			$1::text,
			unnest($2::bigint[]),
			unnest($3::text[]),
			unnest($4::bigint[]),
			unnest($5::text[]),
			unnest($6::text[]),
			unnest($7::bigint[])
	`
	_, err = pg.FromContext(ctx).Exec(q,
		hash.String(),
		outs.index,
		outs.assetID,
		outs.amount,
		outs.bucketID,
		outs.walletID,
		outs.aIndex,
	)
	return insert, errors.Wrap(err)
}

func initAddrInfoFromRecs(hash bc.Hash, txouts []*bc.TxOutput, recs []*utxodb.Receiver) ([]*appdb.UTXO, error) {
	insert := make([]*appdb.UTXO, len(txouts))
	for i, txo := range txouts {
		addr, err := txscript.PkScriptAddr(txo.Script)
		if err != nil {
			return nil, errors.Wrap(err, "bad pk script")
		}
		u := &appdb.UTXO{
			Addr: addr.String(),
			UTXO: &utxodb.UTXO{
				AssetID:  txo.AssetID.String(),
				Amount:   uint64(txo.Value),
				Outpoint: bc.Outpoint{Hash: hash, Index: uint32(i)},
			},
		}
		if rec := recs[i]; rec != nil {
			u.WalletID = rec.WalletID
			u.BucketID = rec.BucketID
			copy(u.AddrIndex[:], rec.AddrIndex)
			u.IsChange = rec.IsChange
		}
		insert[i] = u
	}
	return insert, nil
}

// loadAddrInfoFromDB loads bucket ID and addr index
// from the addresses table for utxos that need it.
// Not all are guaranteed to be in the database;
// some outputs will be owned by third parties.
// This function loads what it can.
func loadAddrInfoFromDB(ctx context.Context, utxos []*appdb.UTXO) error {
	var addrs []string
	for _, u := range utxos {
		if u.BucketID == "" {
			addrs = append(addrs, u.Addr)
		}
	}

	const q = `
		SELECT address, account_id, manager_node_id, key_index(key_index), is_change
		FROM addresses
		WHERE address IN (SELECT unnest($1::text[]))
	`
	rows, err := pg.FromContext(ctx).Query(q, pg.Strings(addrs))
	if err != nil {
		return errors.Wrap(err, "select")
	}
	defer rows.Close()
	for rows.Next() {
		var (
			addr      string
			walletID  string
			bucketID  string
			addrIndex []uint32
			isChange  bool
		)
		err = rows.Scan(
			&addr,
			&bucketID,
			&walletID,
			(*pg.Uint32s)(&addrIndex),
			&isChange,
		)
		if err != nil {
			return errors.Wrap(err, "scan")
		}
		for _, u := range utxos {
			if u.BucketID == "" && u.Addr == addr {
				u.WalletID = walletID
				u.BucketID = bucketID
				u.IsChange = isChange
				copy(u.AddrIndex[:], addrIndex)
			}
		}
	}
	return errors.Wrap(rows.Err(), "rows")
}

func toKeyIndex(i []uint32) int64 {
	return int64(i[0])<<31 | int64(i[1]&0x7fffffff)
}
