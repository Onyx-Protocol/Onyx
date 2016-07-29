package core

import (
	"testing"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/asset/assettest"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/authn"
)

type fixtureInfo struct {
	u1ID, u2ID       string
	proj1ID, proj2ID string
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

func TestIssuerAuthz(t *testing.T) {
	withCommonFixture(t, func(ctx context.Context, fixtureInfo *fixtureInfo) {
		in1ID := assettest.CreateIssuerNodeFixture(ctx, t, fixtureInfo.proj1ID, "", nil, nil)
		in2ID := assettest.CreateIssuerNodeFixture(ctx, t, fixtureInfo.proj2ID, "", nil, nil)

		cases := []struct {
			userID  string
			inodeID string
			want    error
		}{
			{fixtureInfo.u2ID, in1ID, nil}, {fixtureInfo.u2ID, in2ID, nil},
		}

		for _, c := range cases {
			ctx := authn.NewContext(ctx, c.userID)
			got := issuerAuthz(ctx, c.inodeID)
			if errors.Root(got) != c.want {
				t.Errorf("issuerAuthz(%s, %v) = %q want %q", c.userID, c.inodeID, got, c.want)
			}
		}
	})
}

func TestAssetAuthz(t *testing.T) {
	withCommonFixture(t, func(ctx context.Context, fixtureInfo *fixtureInfo) {
		in1ID := assettest.CreateIssuerNodeFixture(ctx, t, fixtureInfo.proj1ID, "", nil, nil)
		in2ID := assettest.CreateIssuerNodeFixture(ctx, t, fixtureInfo.proj2ID, "", nil, nil)

		a1ID := assettest.CreateAssetFixture(ctx, t, in1ID, "", "")
		a2ID := assettest.CreateAssetFixture(ctx, t, in2ID, "", "")
		a3ID := assettest.CreateAssetFixture(ctx, t, in2ID, "", "")
		err := appdb.ArchiveAsset(ctx, a3ID.String())
		if err != nil {
			panic(err)
		}

		cases := []struct {
			userID  string
			assetID bc.AssetID
			want    error
		}{
			{fixtureInfo.u2ID, a1ID, nil}, {fixtureInfo.u2ID, a2ID, nil}, {fixtureInfo.u2ID, a3ID, appdb.ErrArchived},
		}

		for _, c := range cases {
			ctx := authn.NewContext(ctx, c.userID)
			got := assetAuthz(ctx, c.assetID.String())
			if errors.Root(got) != c.want {
				t.Errorf("assetAuthz(%s, %v) = %q want %q", c.userID, c.assetID, got, c.want)
			}
		}
	})
}

func TestBuildAuthz(t *testing.T) {
	withCommonFixture(t, func(ctx context.Context, fixtureInfo *fixtureInfo) {
		acc1ID := assettest.CreateAccountFixture(ctx, t, nil, 0)
		acc2ID := assettest.CreateAccountFixture(ctx, t, nil, 0)

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

	fixtureInfo.proj1ID = assettest.CreateProjectFixture(ctx, t, "")
	fixtureInfo.proj2ID = assettest.CreateProjectFixture(ctx, t, "")

	fn(ctx, &fixtureInfo)
}
