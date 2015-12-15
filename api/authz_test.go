package api

import (
	"testing"

	"chain/api/asset"
	"chain/api/utxodb"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/authn"
)

var authzFixture = `
	INSERT INTO users(id, email, password_hash)
		VALUES ('u1', 'u1', ''), ('u2', 'u2', '');
	INSERT INTO projects(id, name)
		VALUES ('proj1', 'proj1'), ('proj2', 'proj2'), ('proj3', 'proj3');
	INSERT INTO members (project_id, user_id, role)
	VALUES
		('proj1', 'u1', 'admin'),
		('proj1', 'u2', 'developer'),
		('proj2', 'u1', 'admin'),
		('proj2', 'u2', 'admin');
`

func TestProjectAdminAuthz(t *testing.T) {
	ctx := pgtest.NewContext(t, authzFixture)
	defer pgtest.Finish(ctx)

	cases := []struct {
		userID string
		projID string
		want   error
	}{
		{"u1", "proj1", nil},         // admin
		{"u2", "proj1", errNotAdmin}, // not an admin
		{"u3", "proj1", errNotAdmin}, // not a member
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
	ctx := pgtest.NewContext(t, authzFixture)
	defer pgtest.Finish(ctx)

	cases := []struct {
		userID string
		projID []string
		want   error
	}{
		{"u1", []string{"proj1"}, nil},                            // admin
		{"u2", []string{"proj1"}, nil},                            // member
		{"u3", []string{"proj1"}, errNoAccessToResource},          // not a member
		{"u1", []string{"proj1", "proj2"}, errNoAccessToResource}, // two projects
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
	ctx := pgtest.NewContext(t, authzFixture, `
		INSERT INTO manager_nodes (id, project_id, label)
			VALUES ('mn1', 'proj1', 'x'), ('mn2', 'proj2', 'x'), ('mn3', 'proj3', 'x');
	`)
	defer pgtest.Finish(ctx)

	cases := []struct {
		userID        string
		managerNodeID string
		want          error
	}{
		{"u2", "mn1", nil}, {"u2", "mn2", nil}, {"u2", "mn3", errNoAccessToResource},
	}

	for _, c := range cases {
		ctx := authn.NewContext(ctx, c.userID)
		got := managerAuthz(ctx, c.managerNodeID)
		if errors.Root(got) != c.want {
			t.Errorf("managerAuthz(%s, %v) = %q want %q", c.userID, c.managerNodeID, got, c.want)
		}
	}
}

func TestAccountAuthz(t *testing.T) {
	ctx := pgtest.NewContext(t, authzFixture, `
		INSERT INTO manager_nodes (id, project_id, label)
			VALUES ('mn1', 'proj1', 'x'), ('mn2', 'proj2', 'x'), ('mn3', 'proj3', 'x');
		INSERT INTO accounts (id, manager_node_id, key_index)
			VALUES ('acc1', 'mn1', 0), ('acc2', 'mn2', 0), ('acc3', 'mn3', 0);
	`)
	defer pgtest.Finish(ctx)

	cases := []struct {
		userID    string
		accountID string
		want      error
	}{
		{"u2", "acc1", nil}, {"u2", "acc2", nil}, {"u2", "acc3", errNoAccessToResource},
	}

	for _, c := range cases {
		ctx := authn.NewContext(ctx, c.userID)
		got := accountAuthz(ctx, c.accountID)
		if errors.Root(got) != c.want {
			t.Errorf("accountAuthz(%s, %v) = %q want %q", c.userID, c.accountID, got, c.want)
		}
	}
}

func TestIssuerAuthz(t *testing.T) {
	ctx := pgtest.NewContext(t, authzFixture, `
		INSERT INTO issuer_nodes (id, project_id, label, keyset)
			VALUES ('in1', 'proj1', 'x', '{}'), ('in2', 'proj2', 'x', '{}'), ('in3', 'proj3', 'x', '{}');
	`)
	defer pgtest.Finish(ctx)

	cases := []struct {
		userID  string
		inodeID string
		want    error
	}{
		{"u2", "in1", nil}, {"u2", "in2", nil}, {"u2", "in3", errNoAccessToResource},
	}

	for _, c := range cases {
		ctx := authn.NewContext(ctx, c.userID)
		got := issuerAuthz(ctx, c.inodeID)
		if errors.Root(got) != c.want {
			t.Errorf("issuerAuthz(%s, %v) = %q want %q", c.userID, c.inodeID, got, c.want)
		}
	}
}

func TestAssetAuthz(t *testing.T) {
	ctx := pgtest.NewContext(t, authzFixture, `
		INSERT INTO issuer_nodes (id, project_id, label, keyset)
			VALUES ('in1', 'proj1', 'x', '{}'), ('in2', 'proj2', 'x', '{}'), ('in3', 'proj3', 'x', '{}');
		INSERT INTO assets (id, issuer_node_id, key_index, redeem_script, issuance_script, label)
		VALUES
			('a1', 'in1', 0, '\x'::bytea, '\x'::bytea, ''),
			('a2', 'in2', 0, '\x'::bytea, '\x'::bytea, ''),
			('a3', 'in3', 0, '\x'::bytea, '\x'::bytea, '');
	`)
	defer pgtest.Finish(ctx)

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
	ctx := pgtest.NewContext(t, authzFixture, `
		INSERT INTO manager_nodes (id, project_id, label)
			VALUES ('mn1', 'proj1', 'x'), ('mn2', 'proj2', 'x'), ('mn3', 'proj3', 'x');
		INSERT INTO accounts (id, manager_node_id, key_index)
			VALUES
				('acc1', 'mn1', 0), ('acc2', 'mn2', 0), ('acc3', 'mn3', 0),
				('acc4', 'mn1', 1), ('acc5', 'mn2', 1), ('acc6', 'mn3', 1);
	`)
	defer pgtest.Finish(ctx)

	cases := []struct {
		userID  string
		request []buildReq
		want    error
	}{
		{
			userID: "u2",
			request: []buildReq{{
				Inputs:  []utxodb.Input{{AccountID: "acc1"}},
				Outputs: []*asset.Output{{AccountID: "acc4"}},
			}},
			want: nil,
		},
		{
			userID: "u2",
			request: []buildReq{{
				Inputs:  []utxodb.Input{{AccountID: "acc1"}},
				Outputs: []*asset.Output{{AccountID: "acc4"}},
			}, {
				Inputs: []utxodb.Input{{AccountID: "acc4"}},
			}},
			want: nil,
		},
		{
			userID: "u2",
			request: []buildReq{{
				Inputs:  []utxodb.Input{{AccountID: "acc3"}},
				Outputs: []*asset.Output{{AccountID: "acc6"}},
			}},
			want: errNoAccessToResource,
		},
		{
			userID: "u2",
			request: []buildReq{{
				Inputs:  []utxodb.Input{{AccountID: "acc1"}},
				Outputs: []*asset.Output{{AccountID: "acc2"}},
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
