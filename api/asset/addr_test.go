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
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	wallet, err := appdb.InsertWallet(ctx, "proj-id-0", "foo", []*hdkey.XKey{dummyXPub}, nil)
	if err != nil {
		t.Fatal(err)
	}
	bucket, err := appdb.CreateBucket(ctx, wallet.ID, "foo")
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
		WalletID:     wallet.ID,
		WalletIndex:  []uint32{0, 1},
		BucketIndex:  []uint32{0, 0},
		Index:        []uint32{0, 1},
		SigsRequired: 1,
		Keys:         []*hdkey.XKey{dummyXPub},

		Address: "3LkNaCapeRBLcdm5mfH9xv8snvrfzcsixu",
		RedeemScript: []byte{
			81, 33, 2, 241, 154, 202, 111, 123, 48, 123, 116, 244, 53,
			11, 207, 218, 165, 175, 26, 38, 65, 147, 76, 125, 77, 183,
			254, 50, 18, 62, 238, 216, 139, 92, 16, 81, 174,
		},
		PKScript: []byte{
			169, 20, 209, 12, 223, 249, 230, 16, 228, 14, 42, 205, 213, 7,
			90, 164, 51, 115, 60, 99, 212, 242, 135,
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
