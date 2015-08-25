package wallets

import (
	"encoding/hex"
	"reflect"
	"testing"

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

func TestInsertOutputs(t *testing.T) {
	pgtest.ResetWithSQL(t, `
		INSERT INTO wallets (id, application_id, label, current_rotation, pek, key_index)
		VALUES('w1', 'a1', '', 'c1', '', 0);
		INSERT INTO buckets (id, wallet_id, key_index) VALUES('b1', 'w1', 0);
		INSERT INTO receivers (id, bucket_id, wallet_id, address, keyset, key_index)
		VALUES ('r1', 'b1', 'w1', '3H9gBofbYu4uQXwfMVcFiWjQHXf6vmnVGB', '{}', 0);
	`)
	tx := wire.NewMsgTx()

	pkscript, _ := hex.DecodeString("a914a994a46855d8f4442b3a6db863628cc020537f4087")
	asset, _ := wire.NewHash20FromStr("AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo")

	tx.AddTxOut(wire.NewTxOut(asset, 1000, pkscript))

	err := InsertOutputs(tx)
	if err != nil {
		t.Fatal("unexptected error:", err)
	}

	const check = `
		SELECT txid, index, asset_id, amount, receiver_id, bucket_id, wallet_id
		FROM outputs
	`
	type output struct {
		txid, assetID, receiverID, bucketID, walletID string
		index                                         uint32
		amount                                        int64
	}
	var got output
	err = db.QueryRow(check).Scan(&got.txid, &got.index, &got.assetID, &got.amount, &got.receiverID, &got.bucketID, &got.walletID)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	want := output{
		txid:       "246c6aa1e5cc2bd1132a37cbc267e2031558aee26a8956e21b749d72920331a7",
		index:      0,
		assetID:    "AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo",
		amount:     1000,
		receiverID: "r1",
		bucketID:   "b1",
		walletID:   "w1",
	}

	if got != want {
		t.Errorf("got output = %+v want %+v", got, want)
	}

	if got := pgtest.Count(t, "outputs"); got != 1 {
		t.Errorf("Count(outputs) = %d want 1", got)
	}
}

func TestTxOutputs(t *testing.T) {
	tx := wire.NewMsgTx()

	pkscript := mustDecodeHex("a914a994a46855d8f4442b3a6db863628cc020537f4087")
	asset, _ := wire.NewHash20FromStr("AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo")

	tx.AddTxOut(wire.NewTxOut(asset, 1000, pkscript))

	got, err := txOutputs(tx)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	want := &outputSet{
		txid:    "246c6aa1e5cc2bd1132a37cbc267e2031558aee26a8956e21b749d72920331a7",
		index:   pg.Uint32s{0},
		assetID: pg.Strings{"AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo"},
		amount:  pg.Int64s{1000},
		addr:    pg.Strings{"3H9gBofbYu4uQXwfMVcFiWjQHXf6vmnVGB"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got txOutputs(tx) = %+v want %+v", got, want)
	}
}

func TestPkScriptAddr(t *testing.T) {
	cases := []struct {
		script string
		want   string
	}{
		{
			script: "a914a994a46855d8f4442b3a6db863628cc020537f4087",
			want:   "3H9gBofbYu4uQXwfMVcFiWjQHXf6vmnVGB",
		},
	}

	for _, c := range cases {
		got, err := pkScriptAddr(mustDecodeHex(c.script))
		if err != nil {
			t.Error("unexptected error", err)
		}
		if got != c.want {
			t.Errorf("got pkScriptAddr(%s) = %v want %v", c.script, got, c.want)
		}
	}
}
