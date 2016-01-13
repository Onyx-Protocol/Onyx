package api

import (
	"testing"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset/assettest"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/net/http/authn"
)

type fixtureInfo struct {
	u1ID, u2ID, u3ID          string
	proj1ID, proj2ID, proj3ID string
}

func TestProjectAdminAuthz(t *testing.T) {
	withCommonFixture(t, func(ctx context.Context, fixtureInfo *fixtureInfo) {
		cases := []struct {
			userID string
			projID string
			want   error
		}{
			{fixtureInfo.u1ID, fixtureInfo.proj1ID, nil},         // admin
			{fixtureInfo.u2ID, fixtureInfo.proj1ID, errNotAdmin}, // not an admin
			{fixtureInfo.u3ID, fixtureInfo.proj1ID, errNotAdmin}, // not a member
		}

		for _, c := range cases {
			ctx := authn.NewContext(ctx, c.userID)
			got := projectAdminAuthz(ctx, c.projID)
			if got != c.want {
				t.Errorf("projectAdminAuthz(%s, %s) = %q want %q", c.userID, c.projID, got, c.want)
			}
		}
	})
}

func TestProjectAuthz(t *testing.T) {
	withCommonFixture(t, func(ctx context.Context, fixtureInfo *fixtureInfo) {
		cases := []struct {
			userID string
			projID []string
			want   error
		}{
			{fixtureInfo.u1ID, []string{fixtureInfo.proj1ID}, nil},                                        // admin
			{fixtureInfo.u2ID, []string{fixtureInfo.proj1ID}, nil},                                        // member
			{fixtureInfo.u3ID, []string{fixtureInfo.proj1ID}, errNoAccessToResource},                      // not a member
			{fixtureInfo.u1ID, []string{fixtureInfo.proj1ID, fixtureInfo.proj2ID}, errNoAccessToResource}, // two projects
		}

		for _, c := range cases {
			ctx := authn.NewContext(ctx, c.userID)
			got := projectAuthz(ctx, c.projID...)
			if errors.Root(got) != c.want {
				t.Errorf("projectAuthz(%s, %v) = %q want %q", c.userID, c.projID, got, c.want)
			}
		}
	})
}

func TestManagerAuthz(t *testing.T) {
	withCommonFixture(t, func(ctx context.Context, fixtureInfo *fixtureInfo) {
		mn1ID := assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.proj1ID, "", nil, nil)
		mn2ID := assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.proj2ID, "", nil, nil)
		mn3ID := assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.proj3ID, "", nil, nil)

		cases := []struct {
			userID        string
			managerNodeID string
			want          error
		}{
			{fixtureInfo.u2ID, mn1ID, nil}, {fixtureInfo.u2ID, mn2ID, nil}, {fixtureInfo.u2ID, mn3ID, errNoAccessToResource},
		}

		for _, c := range cases {
			ctx := authn.NewContext(ctx, c.userID)
			got := managerAuthz(ctx, c.managerNodeID)
			if errors.Root(got) != c.want {
				t.Errorf("managerAuthz(%s, %v) = %q want %q", c.userID, c.managerNodeID, got, c.want)
			}
		}
	})
}

func TestAccountAuthz(t *testing.T) {
	withCommonFixture(t, func(ctx context.Context, fixtureInfo *fixtureInfo) {
		mn1ID := assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.proj1ID, "", nil, nil)
		mn2ID := assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.proj2ID, "", nil, nil)
		mn3ID := assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.proj3ID, "", nil, nil)

		acc1ID := assettest.CreateAccountFixture(ctx, t, mn1ID, "", nil)
		acc2ID := assettest.CreateAccountFixture(ctx, t, mn2ID, "", nil)
		acc3ID := assettest.CreateAccountFixture(ctx, t, mn3ID, "", nil)

		cases := []struct {
			userID    string
			accountID string
			want      error
		}{
			{fixtureInfo.u2ID, acc1ID, nil}, {fixtureInfo.u2ID, acc2ID, nil}, {fixtureInfo.u2ID, acc3ID, errNoAccessToResource},
		}

		for _, c := range cases {
			ctx := authn.NewContext(ctx, c.userID)
			got := accountAuthz(ctx, c.accountID)
			if errors.Root(got) != c.want {
				t.Errorf("accountAuthz(%s, %v) = %q want %q", c.userID, c.accountID, got, c.want)
			}
		}
	})
}

func TestIssuerAuthz(t *testing.T) {
	withCommonFixture(t, func(ctx context.Context, fixtureInfo *fixtureInfo) {
		in1ID := assettest.CreateIssuerNodeFixture(ctx, t, fixtureInfo.proj1ID, "", nil, nil)
		in2ID := assettest.CreateIssuerNodeFixture(ctx, t, fixtureInfo.proj2ID, "", nil, nil)
		in3ID := assettest.CreateIssuerNodeFixture(ctx, t, fixtureInfo.proj3ID, "", nil, nil)

		cases := []struct {
			userID  string
			inodeID string
			want    error
		}{
			{fixtureInfo.u2ID, in1ID, nil}, {fixtureInfo.u2ID, in2ID, nil}, {fixtureInfo.u2ID, in3ID, errNoAccessToResource},
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
		in3ID := assettest.CreateIssuerNodeFixture(ctx, t, fixtureInfo.proj3ID, "", nil, nil)

		a1ID := assettest.CreateAssetFixture(ctx, t, in1ID, "")
		a2ID := assettest.CreateAssetFixture(ctx, t, in2ID, "")
		a3ID := assettest.CreateAssetFixture(ctx, t, in3ID, "")

		cases := []struct {
			userID  string
			assetID bc.AssetID
			want    error
		}{
			{fixtureInfo.u2ID, a1ID, nil}, {fixtureInfo.u2ID, a2ID, nil}, {fixtureInfo.u2ID, a3ID, errNoAccessToResource},
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
		mn1ID := assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.proj1ID, "", nil, nil)
		mn2ID := assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.proj2ID, "", nil, nil)
		mn3ID := assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.proj3ID, "", nil, nil)

		acc1ID := assettest.CreateAccountFixture(ctx, t, mn1ID, "", nil)
		acc2ID := assettest.CreateAccountFixture(ctx, t, mn2ID, "", nil)
		// acc3ID := assettest.CreateAccountFixture(ctx, t, mn3ID, "", nil)
		acc4ID := assettest.CreateAccountFixture(ctx, t, mn1ID, "", nil)
		// acc5ID := assettest.CreateAccountFixture(ctx, t, mn2ID, "", nil)
		acc6ID := assettest.CreateAccountFixture(ctx, t, mn3ID, "", nil)

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
								AccountID: acc4ID,
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
								AccountID: acc4ID,
							},
						},
					},
					&BuildRequest{
						Sources: []*Source{
							&Source{
								AssetID:   assetIDPtr,
								AccountID: acc4ID,
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
							&Source{
								AssetID:   assetIDPtr,
								AccountID: acc2ID,
							},
						},
						Dests: []*Destination{
							&Destination{
								AssetID:   assetIDPtr,
								AccountID: acc6ID,
							},
						},
					},
				},
				want: errNoAccessToResource,
			},
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
				want: errNoAccessToResource,
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
	ctx := assettest.NewContextWithGenesisBlock(t)
	defer pgtest.Finish(ctx)

	var (
		fixtureInfo fixtureInfo
		err         error
	)

	fixtureInfo.u1ID = assettest.CreateUserFixture(ctx, t, "", "")
	fixtureInfo.u2ID = assettest.CreateUserFixture(ctx, t, "", "")
	fixtureInfo.u3ID = assettest.CreateUserFixture(ctx, t, "", "")

	fixtureInfo.proj1ID = assettest.CreateProjectFixture(ctx, t, fixtureInfo.u1ID, "")
	err = appdb.AddMember(ctx, fixtureInfo.proj1ID, fixtureInfo.u2ID, "developer")
	if err != nil {
		panic(err)
	}

	fixtureInfo.proj2ID = assettest.CreateProjectFixture(ctx, t, fixtureInfo.u1ID, "")
	err = appdb.AddMember(ctx, fixtureInfo.proj2ID, fixtureInfo.u2ID, "admin")
	if err != nil {
		panic(err)
	}

	fixtureInfo.proj3ID = assettest.CreateProjectFixture(ctx, t, "", "")

	fn(ctx, &fixtureInfo)
}
