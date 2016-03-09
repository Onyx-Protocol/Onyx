package api

import (
	"testing"

	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/net/http/authn"
)

func TestCreateIssuerNode(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	uid := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra")
	proj0 := assettest.CreateProjectFixture(ctx, t, uid, "x")

	ctx = authn.NewContext(ctx, uid)

	req := map[string]interface{}{
		"label":               "node",
		"keys":                []*asset.CreateNodeKeySpec{{Type: "node", Generate: true}},
		"signatures_required": 1,
	}

	_, err := createIssuerNode(ctx, proj0, req)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	var count int
	var checkQ = `SELECT COUNT(*) FROM issuer_nodes`
	err = pg.QueryRow(ctx, checkQ).Scan(&count)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if count != 1 {
		t.Fatalf("checking results, want=1, got=%d", count)
	}
}

func TestCreateIssuerNodeDeprecated(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)
	uid := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra")
	proj0 := assettest.CreateProjectFixture(ctx, t, uid, "x")

	ctx = authn.NewContext(ctx, uid)

	req := map[string]interface{}{
		"label":        "deprecated node",
		"generate_key": true,
	}

	_, err := createIssuerNode(ctx, proj0, req)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	var count int
	var checkQ = `SELECT COUNT(*) FROM issuer_nodes`
	err = pg.QueryRow(ctx, checkQ).Scan(&count)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if count != 1 {
		t.Fatalf("checking results, want=1, got=%d", count)
	}
}
