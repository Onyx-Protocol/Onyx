package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/fedchain-sandbox/hdkey"
)

func TestCreateBucket(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	walletID, err := CreateWallet(ctx, "app-id-0", "foo", []*hdkey.XKey{dummyXPub})
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

	walletID, err := CreateWallet(ctx, "app-id-0", "foo", []*hdkey.XKey{dummyXPub})
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

func TestListBuckets(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO applications (id, name) VALUES
			('app-id-0', 'app-0');

		INSERT INTO wallets (id, application_id, key_index, label) VALUES
			('wallet-id-0', 'app-id-0', 0, 'wallet-0'),
			('wallet-id-1', 'app-id-0', 1, 'wallet-1');

		INSERT INTO buckets (id, wallet_id, key_index, label) VALUES
			('bucket-id-0', 'wallet-id-0', 0, 'bucket-0'),
			('bucket-id-1', 'wallet-id-0', 1, 'bucket-1'),
			('bucket-id-2', 'wallet-id-1', 2, 'bucket-2');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		walletID string
		want     []*Bucket
	}{
		{
			"wallet-id-0",
			[]*Bucket{
				{ID: "bucket-id-0", Label: "bucket-0", Index: []uint32{0, 0}},
				{ID: "bucket-id-1", Label: "bucket-1", Index: []uint32{0, 1}},
			},
		},
		{
			"wallet-id-1",
			[]*Bucket{
				{ID: "bucket-id-2", Label: "bucket-2", Index: []uint32{0, 2}},
			},
		},
		{
			"nonexistent",
			nil,
		},
	}

	for _, ex := range examples {
		got, err := ListBuckets(ctx, ex.walletID)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("buckets in wallet %v:\ngot:  %v\nwant: %v", ex.walletID, got, ex.want)
		}
	}
}
