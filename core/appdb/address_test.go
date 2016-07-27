package appdb_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	. "chain/core/appdb"
	"chain/core/asset/assettest"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestAddressLoadNextIndex(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	ResetSeqs(ctx, t) // Force predictable values.
	mn := assettest.CreateManagerNodeFixture(ctx, t, "", "", nil, nil)
	acc := assettest.CreateAccountFixture(ctx, t, mn, "", nil)

	exp := time.Now().Add(5 * time.Minute)
	addr := &Address{
		AccountID: acc,
		Amount:    100,
		Expires:   exp,
	}
	err := addr.LoadNextIndex(ctx) // get most fields from the db given AccountID
	if err != nil {
		t.Fatal(err)
	}

	want := &Address{
		AccountID: acc,
		Amount:    100,
		Expires:   exp,

		ManagerNodeID:    mn,
		ManagerNodeIndex: []uint32{0, 1},
		AccountIndex:     []uint32{0, 0},
		Index:            []uint32{0, 1},
		SigsRequired:     1,
		Keys:             []*hd25519.XPub{testutil.TestXPub},
	}

	if !reflect.DeepEqual(addr, want) {
		t.Errorf("addr = %+v want %+v", addr, want)
	}
}

func TestAddressInsert(t *testing.T) {
	t0 := time.Now()
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	ResetSeqs(ctx, t) // Force predictable values.
	mn := assettest.CreateManagerNodeFixture(ctx, t, "", "", nil, nil)
	acc := assettest.CreateAccountFixture(ctx, t, mn, "", nil)

	addr := &Address{
		AccountID:        acc,
		Amount:           100,
		Expires:          t0.Add(5 * time.Minute),
		ManagerNodeID:    mn,
		ManagerNodeIndex: []uint32{0, 1},
		AccountIndex:     []uint32{0, 0},
		Index:            []uint32{0, 0},
		SigsRequired:     1,
		Keys:             []*hd25519.XPub{testutil.TestXPub},

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

func TestCreateAddress(t *testing.T) {
	t0 := time.Now()
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	ResetSeqs(ctx, t) // Force predictable values.
	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "foo", []*hd25519.XPub{testutil.TestXPub}, nil)
	acc0 := assettest.CreateAccountFixture(ctx, t, mn0, "foo", nil)

	exp := t0.Add(5 * time.Minute)
	addr := &Address{
		AccountID: acc0,
		Amount:    100,
		Expires:   exp,
	}

	err := CreateAddress(ctx, addr, true)
	if err != nil {
		t.Fatal(err)
	}

	want := &Address{
		AccountID:        acc0,
		Amount:           100,
		Expires:          exp,
		ManagerNodeID:    mn0,
		ManagerNodeIndex: []uint32{0, 1},
		AccountIndex:     []uint32{0, 0},
		Index:            []uint32{0, 1},
		SigsRequired:     1,
		Keys:             []*hd25519.XPub{testutil.TestXPub},

		RedeemScript: []byte{
			81, 32, 145, 4, 99, 242, 165, 102, 205, 231, 173, 30, 202, 60,
			176, 127, 164, 227, 232, 113, 220, 22, 170, 18, 111, 160, 212,
			1, 7, 154, 68, 185, 145, 112, 81, 174,
		},
		PKScript: []byte{
			118, 170, 32, 251, 181, 160, 192, 129, 19, 82, 90, 4, 19, 222,
			81, 180, 133, 153, 189, 154, 134, 109, 209, 156, 20, 45, 31, 20,
			110, 160, 218, 13, 252, 206, 245, 136, 192,
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
