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
	VALUES ('mn1', 'proj-id-0', 'sandbox', 1, 1, 'foo', 'rot1', 0, 1, 1, now(), now());
	INSERT INTO rotations (id, manager_node_id, keyset)
	VALUES ('rot1', 'mn1', '{xpub661MyMwAqRbcFoBSqmqxsAGLAgoLBDHXgZutXooGvHGKXgqPK9HYiVZNoqhGuwzeFW27JBpgZZEabMZhFHkxehJmT8H3AfmfD4zhniw5jcw}');
	INSERT INTO accounts (
		id, manager_node_id, key_index, created_at, updated_at,
		next_address_index, label
	)
	VALUES ('b1', 'mn1', 0, now(), now(), 0, 'foo');
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

		ManagerNodeID:    "mn1",
		ManagerNodeIndex: []uint32{0, 1},
		BucketIndex:      []uint32{0, 0},
		Index:            []uint32{0, 1},
		SigsRequired:     1,
		Keys:             []*hdkey.XKey{dummyXPub},
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
		BucketID:         "b1",
		Amount:           100,
		Expires:          t0.Add(5 * time.Minute),
		IsChange:         false,
		ManagerNodeID:    "mn1",
		ManagerNodeIndex: []uint32{0, 1},
		BucketIndex:      []uint32{0, 0},
		Index:            []uint32{0, 0},
		SigsRequired:     1,
		Keys:             []*hdkey.XKey{dummyXPub},

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
