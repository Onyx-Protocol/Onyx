package mockhsm

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/crypto/ed25519"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/protocol/bc/legacy"
	"chain/testutil"
)

func TestMockHSMChainKDKeys(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	hsm := New(db)
	xpub, err := hsm.XCreate(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	xpub2, err := hsm.XCreate(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("In the face of ignorance and resistance I wrote financial systems into existence")
	sig, err := hsm.XSign(ctx, xpub.XPub, nil, msg)
	if err != nil {
		t.Fatal(err)
	}
	if !xpub.XPub.Verify(msg, sig) {
		t.Error("expected verify to succeed")
	}
	if xpub2.XPub.Verify(msg, sig) {
		t.Error("expected verify with wrong pubkey to fail")
	}
	path := [][]byte{{3, 2, 6, 3, 8, 2, 7}}
	sig, err = hsm.XSign(ctx, xpub2.XPub, path, msg)
	if err != nil {
		t.Fatal(err)
	}
	if xpub2.XPub.Verify(msg, sig) {
		t.Error("expected verify with underived pubkey of sig from derived privkey to fail")
	}
	if !xpub2.XPub.Derive(path).Verify(msg, sig) {
		t.Error("expected verify with derived pubkey of sig from derived privkey to succeed")
	}
	xpubs, _, err := hsm.ListKeys(ctx, nil, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(xpubs) != 2 {
		t.Error("expected 2 entries in the db")
	}
}

func TestMockHSMEd25519Keys(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	hsm := New(db)
	pub, err := hsm.Create(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	pub2, err := hsm.Create(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	bh := legacy.BlockHeader{}
	msg := bh.Hash()
	sig, err := hsm.Sign(ctx, pub.Pub, &bh)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(pub.Pub, msg.Bytes(), sig) {
		t.Error("expected verify to succeed")
	}
	if ed25519.Verify(pub2.Pub, msg.Bytes(), sig) {
		t.Error("expected verify with wrong pubkey to fail")
	}

	pubs, _, err := hsm.ListKeys(ctx, nil, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(pubs) != 0 {
		t.Errorf("expected 0 entries in the db, got %d", len(pubs))
	}
}

func TestKeyWithAlias(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	hsm := New(db)
	xpub, err := hsm.XCreate(ctx, "some-alias")
	if err != nil {
		t.Fatal(err)
	}

	// List keys, no alias filter
	xpubs, _, err := hsm.ListKeys(ctx, nil, "", 100)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(xpubs[0], xpub) {
		t.Fatalf("expected to get %v instead got %v", spew.Sdump(xpub), spew.Sdump(xpubs[0]))
	}

	// List keys, with matching alias filter
	xpubs, _, err = hsm.ListKeys(ctx, []string{"some-alias", "other-alias"}, "", 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(xpubs) != 1 {
		t.Fatalf("list keys with matching filter expected to get 1 instead got %v", len(xpubs))
	}

	if !testutil.DeepEqual(xpubs[0], xpub) {
		t.Fatalf("expected to get %v instead got %v", spew.Sdump(xpub), spew.Sdump(xpubs[0]))
	}

	// List keys, with non-matching alias filter
	xpubs, _, err = hsm.ListKeys(ctx, []string{"other-alias"}, "", 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(xpubs) != 0 {
		t.Fatalf("list keys with matching filter expected to get 0 instead got %v", len(xpubs))
	}

	// check for uniqueness error
	xpub, err = hsm.XCreate(ctx, "some-alias")
	if xpub != nil {
		t.Fatalf("xpub: got %v want nil", xpub)
	}
	if errors.Root(err) != ErrDuplicateKeyAlias {
		t.Fatalf("error return value: got %v want %v", errors.Root(err), ErrDuplicateKeyAlias)
	}
}

func TestKeyWithEmptyAlias(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	hsm := New(db)
	for i := 0; i < 2; i++ {
		_, err := hsm.XCreate(ctx, "")
		if errors.Root(err) != nil {
			t.Fatal(err)
		}
	}
}

func TestKeyOrdering(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	hsm := New(db)

	xpub1, err := hsm.XCreate(ctx, "first-key")
	if err != nil {
		t.Fatal(err)
	}

	xpub2, err := hsm.XCreate(ctx, "second-key")
	if err != nil {
		t.Fatal(err)
	}

	xpubs, _, err := hsm.ListKeys(ctx, nil, "", 100)
	if err != nil {
		t.Fatal(err)
	}

	// Latest key is returned first
	if !testutil.DeepEqual(xpubs[0], xpub2) {
		t.Fatalf("expected to get %v instead got %v", spew.Sdump(xpub2), spew.Sdump(xpubs[0]))
	}

	_, after, err := hsm.ListKeys(ctx, nil, "", 1)
	if err != nil {
		t.Fatal(err)
	}

	xpubs, _, err = hsm.ListKeys(ctx, nil, after, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Older key is returned in second page
	if !testutil.DeepEqual(xpubs[0], xpub1) {
		t.Fatalf("expected to get %v instead got %v", spew.Sdump(xpub1), spew.Sdump(xpubs[0]))
	}
}

func BenchmarkSign(b *testing.B) {
	b.StopTimer()

	_, db := pgtest.NewDB(b, pgtest.SchemaPath)
	ctx := context.Background()
	hsm := New(db)
	xpub, err := hsm.XCreate(ctx, "")
	if err != nil {
		b.Fatal(err)
	}

	msg := []byte("In the face of ignorance and resistance I wrote financial systems into existence")

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := hsm.XSign(ctx, xpub.XPub, nil, msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}
