package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/txdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/txscript"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/metrics"
)

type sqlUTXODB struct{}

func (sqlUTXODB) LoadUTXOs(ctx context.Context, accountID string, assetID bc.AssetID) (resvOuts []*utxodb.UTXO, err error) {
	bcOuts, err := txdb.LoadUTXOs(ctx, accountID, assetID)
	if err != nil {
		return nil, errors.Wrap(err, "load blockchain outputs")
	}
	poolOuts, err := txdb.LoadPoolUTXOs(ctx, accountID, assetID)
	if err != nil {
		return nil, errors.Wrap(err, "load pool outputs")
	}

	var bcOutpoints []bc.Outpoint
	for _, o := range bcOuts {
		bcOutpoints = append(bcOutpoints, o.Outpoint)
	}
	poolView, err := txdb.NewPoolView(ctx, bcOutpoints)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	inBC := make(map[bc.Outpoint]bool)
	for _, o := range bcOuts {
		if !isSpent(ctx, o.Outpoint, poolView) {
			resvOuts = append(resvOuts, o)
			inBC[o.Outpoint] = true
		}
	}
	for _, o := range poolOuts {
		if !inBC[o.Outpoint] {
			resvOuts = append(resvOuts, o)
		}
	}
	return resvOuts, nil
}

func isSpent(ctx context.Context, p bc.Outpoint, v state.ViewReader) bool {
	o := v.Output(ctx, p)
	return o != nil && o.Spent
}

func (sqlUTXODB) SaveReservations(ctx context.Context, utxos []*utxodb.UTXO, exp time.Time) error {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		UPDATE utxos
		SET reserved_until=$3
		WHERE confirmed
		    AND (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
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

// applyTx updates the output set to reflect
// the effects of tx. It deletes consumed utxos
// and inserts newly-created outputs.
// Must be called inside a transaction.
func applyTx(ctx context.Context, tx *bc.Tx, outRecs []*utxodb.Receiver) (deleted []bc.Outpoint, inserted []*txdb.Output, err error) {
	defer metrics.RecordElapsed(time.Now())

	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction

	err = txdb.InsertTx(ctx, tx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "insert into txs")
	}

	err = txdb.InsertPoolTx(ctx, tx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "insert into pool_txs")
	}

	inserted, err = insertUTXOs(ctx, tx.Hash, tx.Outputs, outRecs)
	if err != nil {
		return nil, nil, errors.Wrap(err, "insert outputs")
	}

	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			continue
		}
		deleted = append(deleted, in.Previous)
	}
	err = txdb.InsertPoolInputs(ctx, deleted)
	if err != nil {
		return nil, nil, errors.Wrap(err, "delete")
	}

	return deleted, inserted, err
}

func insertUTXOs(ctx context.Context, hash bc.Hash, txouts []*bc.TxOutput, recs []*utxodb.Receiver) ([]*txdb.Output, error) {
	if len(txouts) != len(recs) {
		return nil, errors.New("length mismatch")
	}
	defer metrics.RecordElapsed(time.Now())

	// This function inserts utxos into the db, and maps
	// them to receiver info (account id and addr index).
	// There are three cases:
	// 1. UTXO pays change or to an "immediate" account receiver.
	//    In this case, we get the receiver info from recs
	//    (which came from the client and was validated
	//    in FinalizeTx).
	// 2. UTXO pays to an address receiver record.
	//    In this case, we get the receiver info from
	//    the addresses table (and eventually delete
	//    the record).
	// 3. UTXO pays to an unknown address.
	//    In this case, there is no receiver info.
	outs := initAddrInfoFromRecs(hash, txouts, recs) // case 1
	err := loadAddrInfoFromDB(ctx, outs)             // case 2
	if err != nil {
		return nil, err
	}

	err = txdb.InsertPoolOutputs(ctx, hash, outs)
	return outs, errors.Wrap(err)
}

func initAddrInfoFromRecs(hash bc.Hash, txouts []*bc.TxOutput, recs []*utxodb.Receiver) []*txdb.Output {
	insert := make([]*txdb.Output, len(txouts))
	for i, txo := range txouts {
		o := &txdb.Output{
			Output: state.Output{
				TxOutput: *txo,
				Outpoint: bc.Outpoint{Hash: hash, Index: uint32(i)},
			},
		}
		if rec := recs[i]; rec != nil {
			o.AccountID = rec.AccountID
			o.ManagerNodeID = rec.ManagerNodeID
			copy(o.AddrIndex[:], rec.AddrIndex)
		}
		insert[i] = o
	}
	return insert
}

// loadAddrInfoFromDB loads account ID, manager node ID, and addr index
// from the addresses table for outputs that need it.
// Not all are guaranteed to be in the database;
// some outputs will be owned by third parties.
// This function loads what it can.
func loadAddrInfoFromDB(ctx context.Context, outs []*txdb.Output) error {
	var (
		addrs      []string
		outsByAddr = make(map[string]*txdb.Output)
	)
	for i, o := range outs {
		if o.AccountID != "" {
			continue
		}

		addr, err := txscript.PkScriptAddr(o.Script)
		if err != nil {
			return errors.Wrapf(err, "bad pk script in output %d", i)
		}

		addrs = append(addrs, addr.String())
		outsByAddr[addr.String()] = o
	}

	const q = `
		SELECT address, account_id, manager_node_id, key_index(key_index)
		FROM addresses
		WHERE address IN (SELECT unnest($1::text[]))
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, pg.Strings(addrs))
	if err != nil {
		return errors.Wrap(err, "select")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			addr          string
			managerNodeID string
			accountID     string
			addrIndex     []uint32
		)
		err = rows.Scan(
			&addr,
			&accountID,
			&managerNodeID,
			(*pg.Uint32s)(&addrIndex),
		)
		if err != nil {
			return errors.Wrap(err, "scan")
		}

		o := outsByAddr[addr]
		o.AccountID = accountID
		o.ManagerNodeID = managerNodeID
		copy(o.AddrIndex[:], addrIndex)
	}

	return errors.Wrap(rows.Err(), "rows")
}
