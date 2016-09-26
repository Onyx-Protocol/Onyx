package mockhsm

import (
	"context"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestMockHSM(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	hsm := New(db)
	xpub, err := hsm.CreateKey(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	xpub2, err := hsm.CreateKey(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("In the face of ignorance and resistance I wrote financial systems into existence")
	sig, err := hsm.Sign(ctx, xpub.XPub, nil, msg)
	if err != nil {
		t.Fatal(err)
	}
	if !xpub.XPub.Verify(msg, sig) {
		t.Error("expected verify to succeed")
	}
	if xpub2.XPub.Verify(msg, sig) {
		t.Error("expected verify with wrong pubkey to fail")
	}
	path := []uint32{3, 2, 6, 3, 8, 2, 7}
	sig, err = hsm.Sign(ctx, xpub2.XPub, path, msg)
	if err != nil {
		t.Fatal(err)
	}
	if xpub2.XPub.Verify(msg, sig) {
		t.Error("expected verify with underived pubkey of sig from derived privkey to fail")
	}
	if !xpub2.XPub.Derive(path).Verify(msg, sig) {
		t.Error("expected verify with derived pubkey of sig from derived privkey to succeed")
	}
	xpubs, _, err := hsm.ListKeys(ctx, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(xpubs) != 2 {
		t.Error("expected 2 entries in the db")
	}
}

func TestKeyWithAlias(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	hsm := New(db)
	xpub, err := hsm.CreateKey(ctx, "some-alias")
	if err != nil {
		t.Fatal(err)
	}
	xpubs, _, err := hsm.ListKeys(ctx, "", 100)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(xpubs[0], xpub) {
		t.Fatalf("expected to get %v instead got %v", spew.Sdump(xpub), spew.Sdump(xpubs[0]))
	}

	// check for uniqueness error
	xpub, err = hsm.CreateKey(ctx, "some-alias")
	if xpub != nil {
		t.Fatalf("xpub: got %v want nil", xpub)
	}
	if errors.Root(err) != ErrDuplicateKeyAlias {
		t.Fatalf("error return value: got %v want %v", errors.Root(err), ErrDuplicateKeyAlias)
	}
}

func TestKeyWithEmptyAlias(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	hsm := New(db)
	for i := 0; i < 2; i++ {
		_, err := hsm.CreateKey(ctx, "")
		if errors.Root(err) != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkSign(b *testing.B) {
	b.StopTimer()

	_, db := pgtest.NewDB(b, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	hsm := New(db)
	xpub, err := hsm.CreateKey(ctx, "")
	if err != nil {
		b.Fatal(err)
	}

	msg := []byte("In the face of ignorance and resistance I wrote financial systems into existence")

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := hsm.Sign(ctx, xpub.XPub, nil, msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}
