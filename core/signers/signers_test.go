package signers

import (
	"context"
	"fmt"
	"testing"

	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/testutil"
)

var dummyXPub = mustDecodeKey("48161b6ca79fe3ae248eaf1a32c66a07db901d81ec3f172b16d3ca8b0de37cd8c49975a24499c5d7a40708f4f13d5445cf87fed54ef5a4a5c47a7689a12e73f9")

func TestCreate(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)

	cases := []struct {
		typ    string
		xpubs  []chainkd.XPub
		quorum int
		want   error
	}{{
		typ:    "account",
		xpubs:  []chainkd.XPub{},
		quorum: 1,
		want:   ErrNoXPubs,
	}, {
		typ:    "account",
		xpubs:  []chainkd.XPub{testutil.TestXPub, testutil.TestXPub},
		quorum: 2,
		want:   ErrDupeXPub,
	}, {
		typ:    "account",
		xpubs:  []chainkd.XPub{testutil.TestXPub},
		quorum: 0,
		want:   ErrBadQuorum,
	}, {
		typ:    "account",
		xpubs:  []chainkd.XPub{testutil.TestXPub},
		quorum: 2,
		want:   ErrBadQuorum,
	}, {
		typ:    "account",
		xpubs:  []chainkd.XPub{testutil.TestXPub},
		quorum: 1,
		want:   nil,
	}, {
		typ: "account",
		xpubs: []chainkd.XPub{
			testutil.TestXPub,
			dummyXPub,
		},
		quorum: 3,
		want:   ErrBadQuorum,
	}, {
		typ: "account",
		xpubs: []chainkd.XPub{
			testutil.TestXPub,
			dummyXPub,
		},
		quorum: 1,
		want:   nil,
	}, {
		typ: "account",
		xpubs: []chainkd.XPub{
			testutil.TestXPub,
			dummyXPub,
		},
		quorum: 2,
		want:   nil,
	}}

	for _, c := range cases {
		_, got := Create(ctx, db, c.typ, c.xpubs, c.quorum, "")

		if errors.Root(got) != c.want {
			t.Errorf("Create(%s, %v, %d) = %q want %q", c.typ, c.xpubs, c.quorum, errors.Root(got), c.want)
		}
	}
}

func TestCreateIdempotency(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)

	clientToken := "test"
	signer, err := Create(
		ctx,
		db,
		"account",
		[]chainkd.XPub{testutil.TestXPub},
		1,
		clientToken,
	)

	if err != nil {
		testutil.FatalErr(t, err)
	}

	signer2, err := Create(
		ctx,
		db,
		"account",
		[]chainkd.XPub{testutil.TestXPub},
		1,
		clientToken,
	)

	if err != nil {
		testutil.FatalErr(t, err)
	}

	if signer.ID != signer2.ID {
		t.Error("expected duplicate Create call to retrieve existing signer")
	}
}

func TestFind(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)

	s1 := createFixture(ctx, db, t)

	cases := []struct {
		typ  string
		id   string
		want error
	}{{
		typ:  "account",
		id:   "nonexistent",
		want: pg.ErrUserInputNotFound,
	}, {
		typ:  "badtype",
		id:   s1.ID,
		want: ErrBadType,
	}, {
		typ:  s1.Type,
		id:   s1.ID,
		want: nil,
	}}

	for _, c := range cases {
		_, got := Find(ctx, db, c.typ, c.id)

		if errors.Root(got) != c.want {
			t.Errorf("Find(%s, %s) = %q want %q", c.typ, c.id, errors.Root(got), c.want)
		}
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)

	var signers []*Signer
	for i := 0; i < 5; i++ {
		signers = append(signers, createFixture(ctx, db, t))
	}

	cases := []struct {
		typ      string
		prev     string
		limit    int
		want     []*Signer
		wantLast string
	}{{
		typ:      "account",
		prev:     "",
		limit:    10,
		want:     signers,
		wantLast: signers[4].ID,
	}, {
		typ:      "account",
		prev:     "",
		limit:    3,
		want:     signers[0:3],
		wantLast: signers[2].ID,
	}, {
		typ:      "account",
		prev:     signers[2].ID,
		limit:    2,
		want:     signers[3:5],
		wantLast: signers[4].ID,
	}}

	for _, c := range cases {
		got, gotLast, err := List(ctx, db, c.typ, c.prev, c.limit)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		if !testutil.DeepEqual(got, c.want) {
			t.Errorf("List(%s, %s, %d)\n\tgot:  %+v\n\twant: %+v", c.typ, c.prev, c.limit, got, c.want)
		}

		if gotLast != c.wantLast {
			t.Errorf("List(%s, %s, %d) last = %s want %s", c.typ, c.prev, c.limit, gotLast, c.wantLast)
		}
	}
}

var clientTokenCounter = createCounter()

func createFixture(ctx context.Context, db pg.DB, t testing.TB) *Signer {
	clientToken := fmt.Sprintf("%d", <-clientTokenCounter)
	signer, err := Create(
		ctx,
		db,
		"account",
		[]chainkd.XPub{testutil.TestXPub},
		1,
		clientToken,
	)

	if err != nil {
		testutil.FatalErr(t, err)
	}

	return signer
}

// Creates an infinite stream of integers counting up from 1
func createCounter() <-chan int {
	result := make(chan int)
	go func() {
		var n int
		for {
			n++
			result <- n
		}
	}()
	return result
}

func mustDecodeKey(h string) chainkd.XPub {
	var xpub chainkd.XPub
	err := xpub.UnmarshalText([]byte(h))
	if err != nil {
		panic(err)
	}
	return xpub
}
