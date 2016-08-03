package mockhsm

import (
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestMockHSM(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	hsm := New(db)
	xpub, err := hsm.CreateKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	xpub2, err := hsm.CreateKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("In the face of ignorance and resistance I wrote financial systems into existence")
	sig, err := hsm.Sign(ctx, xpub, nil, msg)
	if err != nil {
		t.Fatal(err)
	}
	if !xpub.Verify(msg, sig) {
		t.Error("expected verify to succeed")
	}
	if xpub2.Verify(msg, sig) {
		t.Error("expected verify with wrong pubkey to fail")
	}
	path := []uint32{3, 2, 6, 3, 8, 2, 7}
	sig, err = hsm.Sign(ctx, xpub2, path, msg)
	if err != nil {
		t.Fatal(err)
	}
	if xpub2.Verify(msg, sig) {
		t.Error("expected verify with underived pubkey of sig from derived privkey to fail")
	}
	if !xpub2.Derive(path).Verify(msg, sig) {
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
