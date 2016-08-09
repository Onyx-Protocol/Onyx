package core

import (
	"testing"

	"golang.org/x/net/context"

	"chain/core/asset/assettest"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/authn"
)

type fixtureInfo struct {
	u1ID, u2ID string
}

func TestAdminAuthz(t *testing.T) {
	withCommonFixture(t, func(ctx context.Context, fixtureInfo *fixtureInfo) {
		cases := []struct {
			userID string
			want   error
		}{
			{fixtureInfo.u1ID, nil},         // admin
			{fixtureInfo.u2ID, errNotAdmin}, // not an admin
		}

		for _, c := range cases {
			ctx := authn.NewContext(ctx, c.userID)
			got := errors.Root(adminAuthz(ctx))
			if got != c.want {
				t.Errorf("adminAuthz(%s) = %q want %q", c.userID, got, c.want)
			}
		}
	})
}

func withCommonFixture(t *testing.T, fn func(context.Context, *fixtureInfo)) {
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)

	_, _, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	var fixtureInfo fixtureInfo

	fixtureInfo.u1ID = assettest.CreateUserFixture(ctx, t, "", "", "admin")
	fixtureInfo.u2ID = assettest.CreateUserFixture(ctx, t, "", "", "developer")

	fn(ctx, &fixtureInfo)
}
