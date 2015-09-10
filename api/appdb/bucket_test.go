package appdb

import (
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
