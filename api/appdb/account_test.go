package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
)

func TestCreateAccount(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, nil, "foo")
		account, err := CreateAccount(ctx, managerNode.ID, "foo", nil)
		if err != nil {
			t.Error("unexpected error", err)
		}
		if account == nil || account.ID == "" {
			t.Error("got nil account or empty id")
		}
		if account.Label != "foo" {
			t.Errorf("label = %q want foo", account.Label)
		}
	})
}

func TestCreateAccountBadLabel(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, nil, "foo")
		_, err := CreateAccount(ctx, managerNode.ID, "", nil)
		if err == nil {
			t.Error("err = nil, want error")
		}
	})
}

func TestCreateAccountWithKey(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
		keys := []string{"keyo"}
		_, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", keys)
		if err != nil {
			t.Error("unexpected error", err)
		}
	})
}

func TestCreateAccountWithMissingKey(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
		_, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", nil)
		if err == nil {
			t.Error("err = nil, want error")
		}
	})
}

func TestCreateAccountWithTooManyKeys(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
		keys := []string{"keyo", "keya", "keyeeeee"}
		_, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", keys)
		if err == nil {
			t.Error("err = nil, want error")
		}
	})
}

func TestListAccounts(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0');

		INSERT INTO manager_nodes (id, project_id, key_index, label) VALUES
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0'),
			('manager-node-id-1', 'proj-id-0', 1, 'manager-node-1');

		INSERT INTO accounts (id, manager_node_id, key_index, label, archived) VALUES
			('account-id-0', 'manager-node-id-0', 0, 'account-0', false),
			('account-id-1', 'manager-node-id-0', 1, 'account-1', false),
			('account-id-2', 'manager-node-id-1', 2, 'account-2', false),
			('account-id-3', 'manager-node-id-0', 3, 'account-3', false),
			('account-id-4', 'manager-node-id-0', 4, 'account-4', true);
	`
	withContext(t, sql, func(ctx context.Context) {
		examples := []struct {
			managerNodeID string
			prev          string
			limit         int
			want          []*Account
			wantLast      string
		}{
			{
				managerNodeID: "manager-node-id-0",
				limit:         5,
				want: []*Account{
					{ID: "account-id-3", Label: "account-3", Index: []uint32{0, 3}},
					{ID: "account-id-1", Label: "account-1", Index: []uint32{0, 1}},
					{ID: "account-id-0", Label: "account-0", Index: []uint32{0, 0}},
				},
				wantLast: "account-id-0",
			},
			{
				managerNodeID: "manager-node-id-1",
				limit:         5,
				want: []*Account{
					{ID: "account-id-2", Label: "account-2", Index: []uint32{0, 2}},
				},
				wantLast: "account-id-2",
			},
			{
				managerNodeID: "nonexistent",
				want:          nil,
			},
			{
				managerNodeID: "manager-node-id-0",
				limit:         2,
				want: []*Account{
					{ID: "account-id-3", Label: "account-3", Index: []uint32{0, 3}},
					{ID: "account-id-1", Label: "account-1", Index: []uint32{0, 1}},
				},
				wantLast: "account-id-1",
			},
			{
				managerNodeID: "manager-node-id-0",
				limit:         2,
				prev:          "account-id-1",
				want: []*Account{
					{ID: "account-id-0", Label: "account-0", Index: []uint32{0, 0}},
				},
				wantLast: "account-id-0",
			},
			{
				managerNodeID: "manager-node-id-0",
				limit:         2,
				prev:          "account-id-0",
				want:          nil,
				wantLast:      "",
			},
		}

		for _, ex := range examples {
			got, gotLast, err := ListAccounts(ctx, ex.managerNodeID, ex.prev, ex.limit)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("ListAccounts(%v, %v, %d):\ngot:  %v\nwant: %v", ex.managerNodeID, ex.prev, ex.limit, got, ex.want)
			}

			if gotLast != ex.wantLast {
				t.Errorf("ListAccounts(%v, %v, %d):\ngot last:  %v\nwant last: %v",
					ex.managerNodeID, ex.prev, ex.limit, gotLast, ex.wantLast)
			}
		}
	})
}

func TestGetAccount(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0');

		INSERT INTO manager_nodes (id, project_id, key_index, label) VALUES
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0');

		INSERT INTO accounts (id, manager_node_id, key_index, label) VALUES
			('account-id-0', 'manager-node-id-0', 0, 'account-0')
	`
	withContext(t, sql, func(ctx context.Context) {
		examples := []struct {
			id      string
			want    *Account
			wantErr error
		}{
			{
				"account-id-0",
				&Account{ID: "account-id-0", Label: "account-0", Index: []uint32{0, 0}},
				nil,
			},
			{
				"nonexistent",
				nil,
				pg.ErrUserInputNotFound,
			},
		}

		for _, ex := range examples {
			t.Log("Example:", ex.id)

			got, gotErr := GetAccount(ctx, ex.id)

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("account:\ngot:  %v\nwant: %v", got, ex.want)
			}

			if errors.Root(gotErr) != ex.wantErr {
				t.Errorf("get account error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
			}
		}
	})
}

func TestUpdateAccount(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, nil, "foo")
		account, err := CreateAccount(ctx, managerNode.ID, "foo", nil)
		if err != nil {
			t.Error("unexpected error", err)
		}
		if account == nil || account.ID == "" {
			t.Error("got nil account or empty id")
		}
		if account.Label != "foo" {
			t.Errorf("label = %q want foo", account.Label)
		}

		newLabel := "bar"
		err = UpdateAccount(ctx, account.ID, &newLabel)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		account, err = GetAccount(ctx, account.ID)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}
		if account.Label != newLabel {
			t.Errorf("expected %s, got %s", newLabel, account.Label)
		}
	})
}

// Test that calling UpdateManagerNode with no new label is a no-op.
func TestUpdateAccountNoUpdate(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, nil, "foo")
		account, err := CreateAccount(ctx, managerNode.ID, "foo", nil)
		if err != nil {
			t.Fatalf("could not create account: %v", err)
		}
		if account == nil {
			t.Fatal("could not create account (got nil)")
		}
		if account.ID == "" {
			t.Fatal("got empty id when creating account")
		}
		if account.Label != "foo" {
			t.Fatalf("wrong label when creating account, expected foo, got %q", account.Label)
		}

		err = UpdateAccount(ctx, account.ID, nil)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		account, err = GetAccount(ctx, account.ID)
		if err != nil {
			t.Fatalf("could not get account with id %s", account.ID)
		}
		if account.Label != "foo" {
			t.Errorf("expected foo, got %s", account.Label)
		}
	})
}

func TestArchiveAccount(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		account := newTestAccount(t, ctx, nil, "account-1")
		err := ArchiveAccount(ctx, account.ID)
		if err != nil {
			t.Errorf("could not archive account with id %s: %v", account.ID, err)
		}

		var archived bool
		checkQ := `SELECT archived FROM accounts WHERE id = $1`
		err = pg.FromContext(ctx).QueryRow(ctx, checkQ, account.ID).Scan(&archived)

		if !archived {
			t.Errorf("expected account %s to be archived", account.ID)
		}
	})
}
