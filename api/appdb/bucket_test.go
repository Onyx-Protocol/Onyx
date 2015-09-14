package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestCreateBucket(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	walletID, err := CreateWallet(ctx, "app-id-0", "foo", []*Key{dummyXPub})
	if err != nil {
		t.Fatal(err)
	}

	bucket, err := CreateBucket(ctx, walletID, "foo")
	if err != nil {
		t.Error("unexpected error", err)
	}
	if bucket == nil || bucket.ID == "" {
		t.Error("got nil bucket or empty id")
	}
	if bucket.Label != "foo" {
		t.Errorf("label = %q want foo", bucket.Label)
	}
}

func TestCreateBucketBadLabel(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	walletID, err := CreateWallet(ctx, "app-id-0", "foo", []*Key{dummyXPub})
	if err != nil {
		t.Fatal(err)
	}

	_, err = CreateBucket(ctx, walletID, "")
	if err == nil {
		t.Error("err = nil, want error")
	}
}

func TestBucketBalance(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO utxos (txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES ('t0', 0, 'a1', 10, 'add0', 'b0', 'w1'),
		       ('t1', 1, 'a1', 5, 'add0', 'b0', 'w1'),
		       ('t2', 2, 'a2', 20, 'add0', 'b0', 'w1');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	bals, err := BucketBalance(ctx, "b0")
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
