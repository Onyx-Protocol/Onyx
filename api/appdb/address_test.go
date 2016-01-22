package appdb

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/fedchain-sandbox/hdkey"
)

var accountFixture = `
	INSERT INTO manager_nodes (
		id, project_id, block_chain, sigs_required, key_index,
		label, current_rotation, next_asset_index, next_account_index,
		accounts_count, created_at, updated_at
	)
	VALUES ('mn1', 'proj-id-0', 'sandbox', 1, 1, 'foo', 'rot1', 0, 1, 1, now(), now());
	INSERT INTO rotations (id, manager_node_id, keyset)
	VALUES ('rot1', 'mn1', '{` + dummyXPub.String() + `}');
	INSERT INTO accounts (
		id, manager_node_id, key_index, created_at, updated_at,
		next_address_index, label
	)
	VALUES ('acc1', 'mn1', 0, now(), now(), 0, 'foo');
`

func TestAddressLoadNextIndex(t *testing.T) {
	ctx := pgtest.NewContext(t, sampleProjectFixture, accountFixture)
	defer pgtest.Finish(ctx)

	// Force predictable values.
	addrIndexNext, addrIndexCap = 1, 100

	exp := time.Now().Add(5 * time.Minute)
	addr := &Address{
		AccountID: "acc1",
		Amount:    100,
		Expires:   exp,
		IsChange:  false,
	}
	err := addr.LoadNextIndex(ctx) // get most fields from the db given AccountID
	if err != nil {
		t.Fatal(err)
	}

	want := &Address{
		AccountID: "acc1",
		Amount:    100,
		Expires:   exp,
		IsChange:  false,

		ManagerNodeID:    "mn1",
		ManagerNodeIndex: []uint32{0, 1},
		AccountIndex:     []uint32{0, 0},
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
	ctx := pgtest.NewContext(t, sampleProjectFixture, accountFixture)
	defer pgtest.Finish(ctx)

	// Force predictable values.
	addrIndexNext, addrIndexCap = 1, 100

	addr := &Address{
		AccountID:        "acc1",
		Amount:           100,
		Expires:          t0.Add(5 * time.Minute),
		IsChange:         false,
		ManagerNodeID:    "mn1",
		ManagerNodeIndex: []uint32{0, 1},
		AccountIndex:     []uint32{0, 0},
		Index:            []uint32{0, 0},
		SigsRequired:     1,
		Keys:             []*hdkey.XKey{dummyXPub},

		RedeemScript: []byte{},
		PKScript:     []byte{},
	}

	err := addr.Insert(ctx) // get most fields from the db given AccountID
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

var dummyXPub2, _ = hdkey.NewXKey("xpub661MyMwAqRbcFoBSqmqxsAGLAgoLBDHXgZutXooGvHGKXgqPK9HYiVZNoqhGuwzeFW27JBpgZZEabMZhFHkxehJmT8H3AfmfD4zhniw5jcw")

func TestCreateAddress(t *testing.T) {
	t0 := time.Now()
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
	`)
	defer pgtest.Finish(ctx)

	// Force predictable values.
	addrIndexNext, addrIndexCap = 1, 100
	_, err := pg.FromContext(ctx).Exec(ctx, `ALTER SEQUENCE manager_nodes_key_index_seq RESTART`)
	if err != nil {
		t.Fatal(err)
	}

	managerNode, err := InsertManagerNode(ctx, "proj-id-0", "foo", []*hdkey.XKey{dummyXPub2}, nil, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	account, err := CreateAccount(ctx, managerNode.ID, "foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	exp := t0.Add(5 * time.Minute)
	addr := &Address{
		AccountID: account.ID,
		Amount:    100,
		Expires:   exp,
		IsChange:  false,
	}

	err = CreateAddress(ctx, addr, true)
	if err != nil {
		t.Fatal(err)
	}

	want := &Address{
		AccountID:        account.ID,
		Amount:           100,
		Expires:          exp,
		IsChange:         false,
		ManagerNodeID:    managerNode.ID,
		ManagerNodeIndex: []uint32{0, 1},
		AccountIndex:     []uint32{0, 0},
		Index:            []uint32{0, 1},
		SigsRequired:     1,
		Keys:             []*hdkey.XKey{dummyXPub2},

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
