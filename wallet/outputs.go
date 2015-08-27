package wallet

import (
	"chain/database/pg"
	"chain/fedchain/wire"
	"errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"golang.org/x/net/context"
)

type outputSet struct {
	txid    string
	index   pg.Uint32s
	assetID pg.Strings
	amount  pg.Int64s
	addr    pg.Strings
}

// Commit updates the output set to reflect
// the effects of tx. It deletes consumed outputs
// and inserts newly-created outputs.
// Must be called inside a transaction.
func Commit(ctx context.Context, tx *wire.MsgTx) error {
	hash := tx.TxSha()
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	err := insertOutputs(ctx, hash, tx.TxOut)
	if err != nil {
		return err
	}
	return deleteOutputs(ctx, tx.TxIn)
}

func deleteOutputs(ctx context.Context, txins []*wire.TxIn) error {
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
		DELETE FROM outputs
		WHERE (txid, index) IN (TABLE outpoints)
	`
	_, err := pg.FromContext(ctx).Exec(q, pg.Strings(txid), pg.Uint32s(index))
	return err
}

func insertOutputs(ctx context.Context, hash wire.Hash32, txouts []*wire.TxOut) error {
	outs := &outputSet{txid: hash.String()}
	err := addTxOutputs(outs, txouts)
	if err != nil {
		return err
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
			FROM receivers
			INNER JOIN newouts ON address=addr
		)
		INSERT INTO outputs
			(txid, index, asset_id, amount, receiver_id, bucket_id, wallet_id)
		TABLE recouts
	`
	_, err = pg.FromContext(ctx).Exec(q,
		outs.txid,
		outs.index,
		outs.assetID,
		outs.amount,
		outs.addr,
	)
	return err
}

func addTxOutputs(outs *outputSet, txouts []*wire.TxOut) error {
	for i, txo := range txouts {
		outs.index = append(outs.index, uint32(i))
		outs.assetID = append(outs.assetID, txo.AssetID.String())
		outs.amount = append(outs.amount, txo.Value)

		addr, err := pkScriptAddr(txo.PkScript)
		if err != nil {
			return err
		}
		outs.addr = append(outs.addr, addr)
	}

	return nil
}

func pkScriptAddr(pkScript []byte) (string, error) {
	pushed, err := txscript.PushedData(pkScript)
	if err != nil {
		return "", err
	}
	if len(pushed) != 1 || len(pushed[0]) != 20 {
		return "", errors.New("output address is not p2sh")
	}
	addr, err := btcutil.NewAddressScriptHashFromHash(pushed[0], &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}
