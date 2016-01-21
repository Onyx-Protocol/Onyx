package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/txbuilder"
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
func applyTx(ctx context.Context, tx *bc.Tx, outRecs []txbuilder.Receiver) (deleted []bc.Outpoint, inserted []*txdb.Output, err error) {
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

func insertUTXOs(ctx context.Context, hash bc.Hash, txouts []*bc.TxOutput, receivers []txbuilder.Receiver) (inserted []*txdb.Output, err error) {
	if len(txouts) != len(receivers) {
		return nil, errors.New("length mismatch")
	}
	defer metrics.RecordElapsed(time.Now())

	var utxoInserters []txbuilder.UTXOInserter

	for i, txOutput := range txouts {
		receiver := receivers[i]
		outpoint := &bc.Outpoint{
			Hash:  hash,
			Index: uint32(i),
		}
		utxoInserters, err = receiver.AccumulateUTXO(ctx, outpoint, txOutput, utxoInserters)
		if err != nil {
			return nil, errors.Wrap(err, "accumulate utxo inserter")
		}
	}

	for _, utxoInserter := range utxoInserters {
		theseInsertions, err := utxoInserter.InsertUTXOs(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "insert utxos")
		}
		inserted = append(inserted, theseInsertions...)
	}

	return inserted, nil
}
