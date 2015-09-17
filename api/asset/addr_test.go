package asset

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/fedchain-sandbox/hdkey"
)

var dummyXPub, _ = hdkey.NewXKey("xpub661MyMwAqRbcFoBSqmqxsAGLAgoLBDHXgZutXooGvHGKXgqPK9HYiVZNoqhGuwzeFW27JBpgZZEabMZhFHkxehJmT8H3AfmfD4zhniw5jcw")

func TestCreateAddress(t *testing.T) {
	t0 := time.Now()
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO applications (id, name) VALUES ('app-id-0', 'app-0');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	wID, err := appdb.CreateWallet(ctx, "app-id-0", "foo", []*hdkey.XKey{dummyXPub})
	if err != nil {
		t.Fatal(err)
	}
	bucket, err := appdb.CreateBucket(ctx, wID, "foo")
	if err != nil {
		t.Fatal(err)
	}

	exp := t0.Add(5 * time.Minute)
	addr := &appdb.Address{
		BucketID: bucket.ID,
		Amount:   100,
		Expires:  exp,
		IsChange: false,
	}

	err = CreateAddress(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}

	want := &appdb.Address{
		BucketID:     bucket.ID,
		Amount:       100,
		Expires:      exp,
		IsChange:     false,
		WalletID:     wID,
		WalletIndex:  []uint32{0, 1},
		BucketIndex:  []uint32{0, 0},
		Index:        []uint32{0, 0},
		SigsRequired: 1,
		Keys:         []*hdkey.XKey{dummyXPub},

		Address: "3PFuxhkDFhSZdhDDQ8wQWcfPJ4gy9ykyxe",
		RedeemScript: []byte{
			81, 33, 3, 235, 3, 212, 79, 59, 30, 218, 127, 119, 123, 123,
			98, 5, 125, 52, 133, 187, 101, 67, 21, 61, 248, 249, 92, 203,
			104, 221, 206, 84, 174, 39, 68, 81, 174,
		},
		PKScript: []byte{
			169, 20, 236, 147, 100, 7, 0, 193, 35, 66, 88, 87, 12, 59, 28,
			66, 25, 89, 14, 41, 149, 42, 135,
		},
	}

	if !strings.HasPrefix(addr.ID, "a") {
		t.Errorf("ID = %q want prefix 'a'", addr.ID)
	}
	addr.ID = ""
	if addr.Created.Before(t0) {
		t.Errorf("Created = %v want after %v", addr.Created, t0)
	}
	addr.Created = time.Time{}
	if !reflect.DeepEqual(addr, want) {
		t.Errorf("addr = %+v want %+v", addr, want)
	}
}
