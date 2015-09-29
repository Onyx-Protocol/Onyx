package api

import (
	"testing"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/authn"
)

var authzFixture = `
	INSERT INTO users(id, email, password_hash)
		VALUES ('u1', 'u1', ''), ('u2', 'u2', '');
	INSERT INTO projects(id, name)
		VALUES ('app1', 'app1'), ('app2', 'app2'), ('app3', 'app3');
	INSERT INTO members (project_id, user_id, role)
	VALUES
		('app1', 'u1', 'admin'),
		('app1', 'u2', 'developer'),
		('app2', 'u1', 'admin'),
		('app2', 'u2', 'admin');
`

func TestProjectAdminAuthz(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, authzFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	cases := []struct {
		userID string
		projID string
		want   error
	}{
		{"u1", "app1", nil},         // admin
		{"u2", "app1", errNotAdmin}, // not an admin
		{"u3", "app1", errNotAdmin}, // not a member
	}

	for _, c := range cases {
		ctx := authn.NewContext(ctx, c.userID)
		got := projectAdminAuthz(ctx, c.projID)
		if got != c.want {
			t.Errorf("projectAdminAuthz(%s, %s) = %q want %q", c.userID, c.projID, got, c.want)
		}
	}
}

func TestProjectAuthz(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, authzFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	cases := []struct {
		userID string
		projID []string
		want   error
	}{
		{"u1", []string{"app1"}, nil},                           // admin
		{"u2", []string{"app1"}, nil},                           // member
		{"u3", []string{"app1"}, errNoAccessToResource},         // not a member
		{"u1", []string{"app1", "app2"}, errNoAccessToResource}, // two apps
	}

	for _, c := range cases {
		ctx := authn.NewContext(ctx, c.userID)
		got := projectAuthz(ctx, c.projID...)
		if errors.Root(got) != c.want {
			t.Errorf("projectAuthz(%s, %v) = %q want %q", c.userID, c.projID, got, c.want)
		}
	}
}

func TestManagerAuthz(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, authzFixture, `
		INSERT INTO manager_nodes (id, project_id, label)
			VALUES ('w1', 'app1', 'x'), ('w2', 'app2', 'x'), ('w3', 'app3', 'x');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	cases := []struct {
		userID   string
		walletID string
		want     error
	}{
		{"u2", "w1", nil}, {"u2", "w2", nil}, {"u2", "w3", errNoAccessToResource},
	}

	for _, c := range cases {
		ctx := authn.NewContext(ctx, c.userID)
		got := managerAuthz(ctx, c.walletID)
		if errors.Root(got) != c.want {
			t.Errorf("managerAuthz(%s, %v) = %q want %q", c.userID, c.walletID, got, c.want)
		}
	}
}

func TestAccountAuthz(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, authzFixture, `
		INSERT INTO manager_nodes (id, project_id, label)
			VALUES ('w1', 'app1', 'x'), ('w2', 'app2', 'x'), ('w3', 'app3', 'x');
		INSERT INTO accounts (id, manager_node_id, key_index)
			VALUES ('b1', 'w1', 0), ('b2', 'w2', 0), ('b3', 'w3', 0);
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	cases := []struct {
		userID   string
		bucketID string
		want     error
	}{
		{"u2", "b1", nil}, {"u2", "b2", nil}, {"u2", "b3", errNoAccessToResource},
	}

	for _, c := range cases {
		ctx := authn.NewContext(ctx, c.userID)
		got := accountAuthz(ctx, c.bucketID)
		if errors.Root(got) != c.want {
			t.Errorf("accountAuthz(%s, %v) = %q want %q", c.userID, c.bucketID, got, c.want)
		}
	}
}

func TestIssuerAuthz(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, authzFixture, `
		INSERT INTO issuer_nodes (id, project_id, label, keyset)
			VALUES ('ag1', 'app1', 'x', '{}'), ('ag2', 'app2', 'x', '{}'), ('ag3', 'app3', 'x', '{}');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	cases := []struct {
		userID  string
		groupID string
		want    error
	}{
		{"u2", "ag1", nil}, {"u2", "ag2", nil}, {"u2", "ag3", errNoAccessToResource},
	}

	for _, c := range cases {
		ctx := authn.NewContext(ctx, c.userID)
		got := issuerAuthz(ctx, c.groupID)
		if errors.Root(got) != c.want {
			t.Errorf("issuerAuthz(%s, %v) = %q want %q", c.userID, c.groupID, got, c.want)
		}
	}
}

func TestAssetAuthz(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, authzFixture, `
		INSERT INTO issuer_nodes (id, project_id, label, keyset)
			VALUES ('ag1', 'app1', 'x', '{}'), ('ag2', 'app2', 'x', '{}'), ('ag3', 'app3', 'x', '{}');
		INSERT INTO assets (id, issuer_node_id, key_index, redeem_script, label)
		VALUES
			('a1', 'ag1', 0, '', ''),
			('a2', 'ag2', 0, '', ''),
			('a3', 'ag3', 0, '', '');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	cases := []struct {
		userID  string
		assetID string
		want    error
	}{
		{"u2", "a1", nil}, {"u2", "a2", nil}, {"u2", "a3", errNoAccessToResource},
	}

	for _, c := range cases {
		ctx := authn.NewContext(ctx, c.userID)
		got := assetAuthz(ctx, c.assetID)
		if errors.Root(got) != c.want {
			t.Errorf("assetAuthz(%s, %v) = %q want %q", c.userID, c.assetID, got, c.want)
		}
	}
}

func TestBuildAuthz(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, authzFixture, `
		INSERT INTO manager_nodes (id, project_id, label)
			VALUES ('w1', 'app1', 'x'), ('w2', 'app2', 'x'), ('w3', 'app3', 'x');
		INSERT INTO accounts (id, manager_node_id, key_index)
			VALUES
				('b1', 'w1', 0), ('b2', 'w2', 0), ('b3', 'w3', 0),
				('b4', 'w1', 1), ('b5', 'w2', 1), ('b6', 'w3', 1);
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	cases := []struct {
		userID  string
		request []buildReq
		want    error
	}{
		{
			userID: "u2",
			request: []buildReq{{
				Inputs:  []utxodb.Input{{BucketID: "b1"}},
				Outputs: []*asset.Output{{BucketID: "b4"}},
			}},
			want: nil,
		},
		{
			userID: "u2",
			request: []buildReq{{
				Inputs:  []utxodb.Input{{BucketID: "b1"}},
				Outputs: []*asset.Output{{BucketID: "b4"}},
			}, {
				Inputs: []utxodb.Input{{BucketID: "b4"}},
			}},
			want: nil,
		},
		{
			userID: "u2",
			request: []buildReq{{
				Inputs:  []utxodb.Input{{BucketID: "b3"}},
				Outputs: []*asset.Output{{BucketID: "b6"}},
			}},
			want: errNoAccessToResource,
		},
		{
			userID: "u2",
			request: []buildReq{{
				Inputs:  []utxodb.Input{{BucketID: "b1"}},
				Outputs: []*asset.Output{{BucketID: "b2"}},
			}},
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
}
