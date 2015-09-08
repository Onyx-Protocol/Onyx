package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestCreateWallet(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	id, err := CreateWallet(ctx, "a1", "foo", []*Key{dummyXPub})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if id == "" {
		t.Errorf("got empty wallet id")
	}
}

func TestWalletBalance(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO outputs (txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES ('t0', 0, 'a1', 10, 'add0', 'b0', 'w1'),
		       ('t1', 1, 'a1', 5, 'add0', 'b0', 'w1'),
		       ('t2', 2, 'a2', 20, 'add0', 'b1', 'w1');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	bals, err := WalletBalance(ctx, "w1")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	want := []*Balance{
		{
			AssetID:   "a1",
			Confirmed: 15,
			Total:     15,
		},
		{
			AssetID:   "a2",
			Confirmed: 20,
			Total:     20,
		},
	}

	if !reflect.DeepEqual(want, bals) {
		t.Errorf("got=%v want=%v", bals, want)
	}
}
