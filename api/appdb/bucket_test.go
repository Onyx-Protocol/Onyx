package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
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

	cases := []struct {
		bID      string
		prev     string
		limit    int
		want     []*Balance
		wantLast string
	}{{
		bID:      "b0",
		limit:    5,
		want:     []*Balance{{"a1", 15, 15}, {"a2", 20, 20}},
		wantLast: "a2",
	}, {
		bID:      "b0",
		prev:     "a1",
		limit:    5,
		want:     []*Balance{{"a2", 20, 20}},
		wantLast: "a2",
	}, {
		bID:      "b0",
		prev:     "a2",
		limit:    5,
		want:     nil,
		wantLast: "",
	}, {
		bID:      "b0",
		limit:    1,
		want:     []*Balance{{"a1", 15, 15}},
		wantLast: "a1",
	}, {
		bID:      "nonexistent",
		limit:    5,
		want:     nil,
		wantLast: "",
	}}

	for _, c := range cases {
		got, gotLast, err := BucketBalance(ctx, c.bID, c.prev, c.limit)
		if err != nil {
			t.Errorf("BucketBalance(%s, %s, %d): unexpected error %v", c.bID, c.prev, c.limit, err)
			continue
		}

		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("BucketBalance(%s, %s, %d) = %v want %v", c.bID, c.prev, c.limit, got, c.want)
		}

		if gotLast != c.wantLast {
			t.Errorf("BucketBalance(%s, %s, %d) last = %v want %v", c.bID, c.prev, c.limit, gotLast, c.wantLast)
		}
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
			('bucket-id-2', 'wallet-id-1', 2, 'bucket-2'),
			('bucket-id-3', 'wallet-id-0', 3, 'bucket-3');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		walletID string
		prev     string
		limit    int
		want     []*Bucket
		wantLast string
	}{
		{
			walletID: "wallet-id-0",
			limit:    5,
			want: []*Bucket{
				{ID: "bucket-id-3", Label: "bucket-3", Index: []uint32{0, 3}},
				{ID: "bucket-id-1", Label: "bucket-1", Index: []uint32{0, 1}},
				{ID: "bucket-id-0", Label: "bucket-0", Index: []uint32{0, 0}},
			},
			wantLast: "bucket-id-0",
		},
		{
			walletID: "wallet-id-1",
			limit:    5,
			want: []*Bucket{
				{ID: "bucket-id-2", Label: "bucket-2", Index: []uint32{0, 2}},
			},
			wantLast: "bucket-id-2",
		},
		{
			walletID: "nonexistent",
			want:     nil,
		},
		{
			walletID: "wallet-id-0",
			limit:    2,
			want: []*Bucket{
				{ID: "bucket-id-3", Label: "bucket-3", Index: []uint32{0, 3}},
				{ID: "bucket-id-1", Label: "bucket-1", Index: []uint32{0, 1}},
			},
			wantLast: "bucket-id-1",
		},
		{
			walletID: "wallet-id-0",
			limit:    2,
			prev:     "bucket-id-1",
			want: []*Bucket{
				{ID: "bucket-id-0", Label: "bucket-0", Index: []uint32{0, 0}},
			},
			wantLast: "bucket-id-0",
		},
		{
			walletID: "wallet-id-0",
			limit:    2,
			prev:     "bucket-id-0",
			want:     nil,
			wantLast: "",
		},
	}

	for _, ex := range examples {
		got, gotLast, err := ListBuckets(ctx, ex.walletID, ex.prev, ex.limit)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("ListBuckets(%v, %v, %d):\ngot:  %v\nwant: %v", ex.walletID, ex.prev, ex.limit, got, ex.want)
		}

		if gotLast != ex.wantLast {
			t.Errorf("ListBuckets(%v, %v, %d):\ngot last:  %v\nwant last: %v",
				ex.walletID, ex.prev, ex.limit, gotLast, ex.wantLast)
		}
	}
}

func TestGetBucket(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO applications (id, name) VALUES
			('app-id-0', 'app-0');

		INSERT INTO wallets (id, application_id, key_index, label) VALUES
			('wallet-id-0', 'app-id-0', 0, 'wallet-0');

		INSERT INTO buckets (id, wallet_id, key_index, label) VALUES
			('bucket-id-0', 'wallet-id-0', 0, 'bucket-0')
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		id      string
		want    *Bucket
		wantErr error
	}{
		{
			"bucket-id-0",
			&Bucket{ID: "bucket-id-0", Label: "bucket-0", Index: []uint32{0, 0}},
			nil,
		},
		{
			"nonexistent",
			nil,
			pg.ErrUserInputNotFound,
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.id)

		got, gotErr := GetBucket(ctx, ex.id)

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("bucket:\ngot:  %v\nwant: %v", got, ex.want)
		}

		if errors.Root(gotErr) != ex.wantErr {
			t.Errorf("get bucket error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
		}
	}
}
