package core

import (
	"testing"

	"golang.org/x/net/context"

	"chain/core/asset/assettest"
	"chain/cos/bc"
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

func TestBuildAuthz(t *testing.T) {
	withCommonFixture(t, func(ctx context.Context, fixtureInfo *fixtureInfo) {
		acc1ID := assettest.CreateAccountFixture(ctx, t, nil, 0, nil)
		acc2ID := assettest.CreateAccountFixture(ctx, t, nil, 0, nil)

		assetIDPtr := &bc.AssetID{}

		cases := []struct {
			userID  string
			request []*BuildRequest
			want    error
		}{
			{
				userID: fixtureInfo.u2ID,
				request: []*BuildRequest{
					&BuildRequest{
						Sources: []*Source{
							&Source{
								AssetID:   assetIDPtr,
								AccountID: acc1ID,
							},
						},
						Dests: []*Destination{
							&Destination{
								AssetID:   assetIDPtr,
								AccountID: acc2ID,
							},
						},
					},
				},
				want: nil,
			},
			{
				userID: fixtureInfo.u2ID,
				request: []*BuildRequest{
					&BuildRequest{
						Sources: []*Source{
							{
								AssetID:   assetIDPtr,
								AccountID: acc1ID,
							},
						},
						Dests: []*Destination{
							&Destination{
								AssetID:   assetIDPtr,
								AccountID: acc2ID,
							},
						},
					},
					&BuildRequest{
						Sources: []*Source{
							&Source{
								AssetID:   assetIDPtr,
								AccountID: acc2ID,
							},
						},
					},
				},
				want: nil,
			},
		}

		for i, c := range cases {
			ctx := authn.NewContext(ctx, c.userID)
			got := buildAuthz(ctx, c.request...)
			if errors.Root(got) != c.want {
				t.Errorf("%d: buildAuthz = %q want %q", i, got, c.want)
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
