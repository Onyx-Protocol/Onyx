package appdb

import (
	"encoding/hex"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/fedchain/wire"
)

func mustDecodeHex(h string) []byte {
	bits, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	return bits
}

func TestInsertUTXOs(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, `
		INSERT INTO wallets (id, application_id, label, current_rotation, key_index)
		VALUES('w1', 'app-id-0', '', 'c1', 0);
		INSERT INTO buckets (id, wallet_id, key_index) VALUES('b1', 'w1', 0);
		INSERT INTO addresses (id, bucket_id, wallet_id, redeem_script, address, pk_script, keyset, key_index)
		VALUES ('a1', 'b1', 'w1', '', '3H9gBofbYu4uQXwfMVcFiWjQHXf6vmnVGB', '', '{}', 0);
	`)
	defer dbtx.Rollback()

	tx := wire.NewMsgTx()

	pkscript, _ := hex.DecodeString("a914a994a46855d8f4442b3a6db863628cc020537f4087")
	asset, _ := wire.NewHash20FromStr("AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo")

	tx.AddTxOut(wire.NewTxOut(asset, 1000, pkscript))

	bgctx := pg.NewContext(context.Background(), dbtx)
	err := insertUTXOs(bgctx, tx.TxSha(), tx.TxOut)
	if err != nil {
		t.Fatal("unexptected error:", err)
	}

	const check = `
		SELECT txid, index, asset_id, amount, address_id, bucket_id, wallet_id
		FROM utxos
	`
	type output struct {
		txid, assetID, addressID, bucketID, walletID string
		index                                        uint32
		amount                                       int64
	}
	var got output
	err = dbtx.QueryRow(check).Scan(&got.txid, &got.index, &got.assetID, &got.amount, &got.addressID, &got.bucketID, &got.walletID)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	want := output{
		txid:      "246c6aa1e5cc2bd1132a37cbc267e2031558aee26a8956e21b749d72920331a7",
		index:     0,
		assetID:   "AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo",
		amount:    1000,
		addressID: "a1",
		bucketID:  "b1",
		walletID:  "w1",
	}

	if got != want {
		t.Errorf("got output = %+v want %+v", got, want)
	}

	if got := pgtest.Count(t, dbtx, "utxos"); got != 1 {
		t.Errorf("Count(utxos) = %d want 1", got)
	}
}

func TestTxOutputs(t *testing.T) {
	tx := wire.NewMsgTx()

	pkscript := mustDecodeHex("a914a994a46855d8f4442b3a6db863628cc020537f4087")
	asset, _ := wire.NewHash20FromStr("AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo")

	tx.AddTxOut(wire.NewTxOut(asset, 1000, pkscript))

	got := new(outputSet)
	err := addTxOutputs(got, tx.TxOut)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	want := &outputSet{
		index:   pg.Uint32s{0},
		assetID: pg.Strings{"AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo"},
		amount:  pg.Int64s{1000},
		addr:    pg.Strings{"3H9gBofbYu4uQXwfMVcFiWjQHXf6vmnVGB"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got txOutputs(tx) = %+v want %+v", got, want)
	}
}
