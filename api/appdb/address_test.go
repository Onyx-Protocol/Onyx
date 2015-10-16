package appdb

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/fedchain-sandbox/hdkey"
)

const bucketFixture = `
	INSERT INTO manager_nodes (
		id, project_id, block_chain, sigs_required, key_index,
		label, current_rotation, next_asset_index, next_account_index,
		accounts_count, created_at, updated_at
	)
	VALUES ('w1', 'proj-id-0', 'sandbox', 1, 1, 'foo', 'rot1', 0, 1, 1, now(), now());
	INSERT INTO rotations (id, manager_node_id, keyset)
	VALUES ('rot1', 'w1', '{xpub661MyMwAqRbcFoBSqmqxsAGLAgoLBDHXgZutXooGvHGKXgqPK9HYiVZNoqhGuwzeFW27JBpgZZEabMZhFHkxehJmT8H3AfmfD4zhniw5jcw}');
	INSERT INTO accounts (
		id, manager_node_id, key_index, created_at, updated_at,
		next_address_index, label
	)
	VALUES ('b1', 'w1', 0, now(), now(), 0, 'foo');
`

func TestAddressLoadNextIndex(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleProjectFixture, bucketFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	exp := time.Now().Add(5 * time.Minute)
	addr := &Address{
		BucketID: "b1",
		Amount:   100,
		Expires:  exp,
		IsChange: false,
	}
	err := addr.LoadNextIndex(ctx) // get most fields from the db given BucketID
	if err != nil {
		t.Fatal(err)
	}

	want := &Address{
		BucketID: "b1",
		Amount:   100,
		Expires:  exp,
		IsChange: false,

		WalletID:     "w1",
		WalletIndex:  []uint32{0, 1},
		BucketIndex:  []uint32{0, 0},
		Index:        []uint32{0, 1},
		SigsRequired: 1,
		Keys:         []*hdkey.XKey{dummyXPub},
	}

	if !reflect.DeepEqual(addr, want) {
		t.Errorf("addr = %+v want %+v", addr, want)
	}
}

func TestAddressInsert(t *testing.T) {
	t0 := time.Now()
	dbtx := pgtest.TxWithSQL(t, sampleProjectFixture, bucketFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	addr := &Address{
		BucketID:     "b1",
		Amount:       100,
		Expires:      t0.Add(5 * time.Minute),
		IsChange:     false,
		WalletID:     "w1",
		WalletIndex:  []uint32{0, 1},
		BucketIndex:  []uint32{0, 0},
		Index:        []uint32{0, 0},
		SigsRequired: 1,
		Keys:         []*hdkey.XKey{dummyXPub},

		Address:      "3abc",
		RedeemScript: []byte{},
		PKScript:     []byte{},
	}

	err := addr.Insert(ctx) // get most fields from the db given BucketID
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(addr.ID, "a") {
		t.Errorf("ID = %q want prefix 'a'", addr.ID)
	}
	if addr.Created.Before(t0) {
		t.Errorf("Created = %v want after %v", addr.Created, t0)
	}
}

func TestAddressesByID(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleProjectFixture, `
		INSERT INTO manager_nodes (id, project_id, label) VALUES('w1', 'proj-id-0', 'w1');
		INSERT INTO accounts (id, manager_node_id, key_index) VALUES('b1', 'w1', 0);
		INSERT INTO addresses (id, manager_node_id, account_id, keyset, key_index, address, redeem_script, pk_script)
		VALUES('a1', 'w1', 'b1', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}', 0, 'a1', '', '');
	`)
	defer dbtx.Rollback()

	ctx := pg.NewContext(context.Background(), dbtx)
	got, err := AddressesByID(ctx, []string{"a1"})
	if err != nil {
		t.Fatal(err)
	}

	k, _ := hdkey.NewXKey("xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd")
	want := &Address{
		ID:           "a1",
		WalletID:     "w1",
		SigsRequired: 1,
		RedeemScript: []byte{},
		WalletIndex:  []uint32{0, 1},
		BucketIndex:  []uint32{0, 0},
		Index:        []uint32{0, 0},
		Keys:         []*hdkey.XKey{k},
	}

	if !reflect.DeepEqual(got[0], want) {
		t.Errorf("got AddressesByID[0] = %v want %v", got[0], want)
	}
}

func TestAddressesByIDMissing(t *testing.T) {
	ctx := pg.NewContext(context.Background(), db)
	_, err := AddressesByID(ctx, []string{"a1"})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "missing address") {
		t.Error("expected missing address error")
	}
}
