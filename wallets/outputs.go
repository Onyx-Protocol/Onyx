package wallets

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

func InsertOutputs(ctx context.Context, tx *wire.MsgTx) error {
	outs, err := txOutputs(tx)
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

func txOutputs(tx *wire.MsgTx) (*outputSet, error) {
	outs := new(outputSet)
	outs.txid = tx.TxSha().String()
	for i, txo := range tx.TxOut {
		outs.index = append(outs.index, uint32(i))
		outs.assetID = append(outs.assetID, txo.AssetID.String())
		outs.amount = append(outs.amount, txo.Value)

		addr, err := pkScriptAddr(txo.PkScript)
		if err != nil {
			return nil, err
		}
		outs.addr = append(outs.addr, addr)
	}

	return outs, nil
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
