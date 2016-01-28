package api

import (
	"net/http"
	"net/url"
	"testing"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/authn"
	"chain/net/http/httpjson"
)

func TestCreateManagerNodeBadXPub(t *testing.T) {
	ctx := pgtest.NewContext(t, testUserFixture, `
		INSERT INTO projects(id, name) VALUES ('a1', 'x');
		INSERT INTO members (project_id, user_id, role)
			VALUES ('a1', 'sample-user-id-0', 'admin');
	`)
	defer pgtest.Finish(ctx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	req := map[string]interface{}{
		"label":               "node",
		"signatures_required": 1,
		"keys":                []*asset.CreateNodeKeySpec{{Type: "node", XPub: "badxpub"}},
	}

	_, err := createManagerNode(ctx, "a1", req)
	if got := errors.Root(err); got != asset.ErrBadKeySpec {
		t.Fatalf("err = %v want %v", got, asset.ErrBadKeySpec)
	}
}

func TestCreateManagerNode(t *testing.T) {
	ctx := pgtest.NewContext(t, testUserFixture, `
		INSERT INTO projects(id, name) VALUES ('a1', 'x');
		INSERT INTO members (project_id, user_id, role)
			VALUES ('a1', 'sample-user-id-0', 'admin');
	`)
	defer pgtest.Finish(ctx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	req := map[string]interface{}{
		"label":               "node",
		"keys":                []*asset.CreateNodeKeySpec{{Type: "node", Generate: true}},
		"signatures_required": 1,
	}

	_, err := createManagerNode(ctx, "a1", req)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	var count int
	var checkQ = `SELECT COUNT(*) FROM manager_nodes`
	err = pg.FromContext(ctx).QueryRow(ctx, checkQ).Scan(&count)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if count != 1 {
		t.Fatalf("checking results, want=1, got=%d", count)
	}
}

func TestCreateManagerNodeDeprecated(t *testing.T) {
	ctx := pgtest.NewContext(t, testUserFixture, `
		INSERT INTO projects(id, name) VALUES ('a1', 'x');
		INSERT INTO members (project_id, user_id, role)
			VALUES ('a1', 'sample-user-id-0', 'admin');
	`)
	defer pgtest.Finish(ctx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	req := map[string]interface{}{
		"label":        "deprecated node",
		"generate_key": true,
	}

	_, err := createManagerNode(ctx, "a1", req)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	var count int
	var checkQ = `SELECT COUNT(*) FROM manager_nodes`
	err = pg.FromContext(ctx).QueryRow(ctx, checkQ).Scan(&count)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if count != 1 {
		t.Fatalf("checking results, want=1, got=%d", count)
	}
}

func TestAccountBalance(t *testing.T) {
	ctx := pgtest.NewContext(t, testUserFixture, `
			INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0');

			INSERT INTO members (project_id, user_id, role) VALUES
				('proj-id-0', 'sample-user-id-0', 'admin');

		INSERT INTO manager_nodes (id, project_id, key_index, label) VALUES
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0');

		INSERT INTO accounts (id, manager_node_id, key_index, label, archived) VALUES
			('account-id-0', 'manager-node-id-0', 0, 'account-0', false),
			('account-id-1', 'manager-node-id-0', 1, 'account-1', true);

		INSERT INTO utxos
			(tx_hash, index, asset_id, amount, addr_index, account_id, manager_node_id, script, confirmed, block_hash, block_height)
		VALUES
			('ctx-0', 0, '0000000000000000000000000000000000000000000000000000000000000000', 1, 0, 'account-id-0', 'manager-node-id-0', '', TRUE, 'bh1', 1),
			('ctx-1', 0, '0000000000000000000000000000000000000000000000000000000000000000', 1, 0, 'account-id-0', 'manager-node-id-0', '', TRUE, 'bh1', 1),
			('ctx-2', 0, '0000000000000000000000000000000000000000000000000000000000000000', 1, 0, 'account-id-1', 'manager-node-id-0', '', TRUE, 'bh1', 1);
	`)
	defer pgtest.Finish(ctx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	cases := []struct {
		accountID string
		wantErr   error
		wantBal   int64
	}{
		{
			accountID: "account-id-0",
			wantErr:   nil,
			wantBal:   2,
		}, {
			accountID: "account-id-1",
			wantErr:   pg.ErrUserInputNotFound,
			wantBal:   0,
		},
	}

	for _, x := range cases {
		// Need to add an http request to the context
		httpURL, err := url.Parse("http://boop.bop/v3/accounts/accountid/balance")
		httpReq := http.Request{URL: httpURL}
		ctx = httpjson.WithRequest(ctx, &httpReq)

		res, err := accountBalance(ctx, x.accountID)

		if errors.Root(err) != x.wantErr {
			t.Fatalf("wanted error=%v, got=%v", x.wantErr, err)
		}

		if err != nil {
			continue
		}

		val, ok := res.(map[string]interface{})
		if !ok {
			t.Fatalf("expected resp to be map[string]interface{}")
		}

		balances, ok := val["balances"].([]*appdb.Balance)
		if !ok {
			t.Fatalf("expected balances to be []*appdb.Balance")
		}

		if balances[0].Total != x.wantBal {
			t.Fatalf("wanted total balance=%d, got=%d", x.wantBal, balances[0].Total)
		}
	}

}
