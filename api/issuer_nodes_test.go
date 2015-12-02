package api

import (
	"testing"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/net/http/authn"
)

func TestCreateIssuerNode(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, testUserFixture, `
		INSERT INTO projects(id, name) VALUES ('a1', 'x');
		INSERT INTO members (project_id, user_id, role)
			VALUES ('a1', 'sample-user-id-0', 'admin');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	req := map[string]interface{}{
		"label":               "node",
		"keys":                []*asset.XPubInit{{Generate: true}},
		"signatures_required": 1,
	}

	_, err := createIssuerNode(ctx, "a1", req)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	var count int
	var checkQ = `
			SELECT COUNT(*) FROM issuer_nodes
		`
	err = pg.FromContext(ctx).QueryRow(checkQ).Scan(&count)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if count != 1 {
		t.Fatalf("checking results, want=1, got=%d", count)
	}
}

func TestCreateIssuerNodeDeprecated(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, testUserFixture, `
		INSERT INTO projects(id, name) VALUES ('a1', 'x');
		INSERT INTO members (project_id, user_id, role)
			VALUES ('a1', 'sample-user-id-0', 'admin');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	req := map[string]interface{}{
		"label":        "deprecated node",
		"generate_key": true,
	}

	_, err := createIssuerNode(ctx, "a1", req)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	var count int
	var checkQ = `
			SELECT COUNT(*) FROM issuer_nodes
		`
	err = pg.FromContext(ctx).QueryRow(checkQ).Scan(&count)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if count != 1 {
		t.Fatalf("checking results, want=1, got=%d", count)
	}
}
