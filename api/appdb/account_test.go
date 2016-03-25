package appdb_test

import (
	"reflect"
	"testing"

	. "chain/api/appdb"
	"chain/api/asset/assettest"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestCreateAccount(t *testing.T) {
	ctx := pgtest.NewContext(t)
	managerNode := newTestManagerNode(t, ctx, nil, "foo")
	account, err := CreateAccount(ctx, managerNode.ID, "foo", nil, nil)
	if err != nil {
		t.Error("unexpected error", err)
	}
	if account == nil || account.ID == "" {
		t.Error("got nil account or empty id")
	}
	if account.Label != "foo" {
		t.Errorf("label = %q want foo", account.Label)
	}
}

func TestCreateAccountBadLabel(t *testing.T) {
	ctx := pgtest.NewContext(t)
	managerNode := newTestManagerNode(t, ctx, nil, "foo")
	_, err := CreateAccount(ctx, managerNode.ID, "", nil, nil)
	if err == nil {
		t.Error("err = nil, want error")
	}
}

func TestCreateAccountWithKey(t *testing.T) {
	ctx := pgtest.NewContext(t)
	managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
	keys := []string{"keyo"}
	_, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", keys, nil)
	if err != nil {
		t.Error("unexpected error", err)
	}
}

func TestCreateAccountWithMissingKey(t *testing.T) {
	ctx := pgtest.NewContext(t)
	managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
	_, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", nil, nil)
	if err == nil {
		t.Error("err = nil, want error")
	}
}

func TestCreateAccountWithTooManyKeys(t *testing.T) {
	ctx := pgtest.NewContext(t)
	managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
	keys := []string{"keyo", "keya", "keyeeeee"}
	_, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", keys, nil)
	if err == nil {
		t.Error("err = nil, want error")
	}
}

func TestCreateAccountIdempotency(t *testing.T) {
	ctx := pgtest.NewContext(t)
	managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
	keys := []string{"keyo"}

	idempotencyKey := "an-idempotency-key-from-the-client"
	acc1, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", keys, &idempotencyKey)
	if err != nil {
		t.Error("unexpected error", err)
	}

	// Re-use the same client token. CreateAccount should not create a new account,
	// and should return acc1 again.
	acc2, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", keys, &idempotencyKey)
	if err != nil {
		t.Error("unexpected error", err)
	}

	if !reflect.DeepEqual(acc1, acc2) {
		t.Errorf("acc2 got=%#v  want=%#v", acc2, acc1)
	}
}

func TestListAccounts(t *testing.T) {
	ctx := pgtest.NewContext(t)

	manager1 := assettest.CreateManagerNodeFixture(ctx, t, "", "m1", nil, nil)
	manager2 := assettest.CreateManagerNodeFixture(ctx, t, "", "m2", nil, nil)
	acc0 := assettest.CreateAccountFixture(ctx, t, manager1, "account-0", nil)
	acc1 := assettest.CreateAccountFixture(ctx, t, manager1, "account-1", nil)
	acc2 := assettest.CreateAccountFixture(ctx, t, manager2, "account-2", nil)
	acc3 := assettest.CreateAccountFixture(ctx, t, manager1, "account-3", nil)
	acc4 := assettest.CreateAccountFixture(ctx, t, manager1, "account-4", nil)

	err := ArchiveAccount(ctx, acc4)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	examples := []struct {
		managerNodeID string
		prev          string
		limit         int
		want          []*Account
		wantLast      string
	}{
		{
			managerNodeID: manager1,
			limit:         5,
			want: []*Account{
				{ID: acc3, Label: "account-3", Index: []uint32{0, 2}},
				{ID: acc1, Label: "account-1", Index: []uint32{0, 1}},
				{ID: acc0, Label: "account-0", Index: []uint32{0, 0}},
			},
			wantLast: acc0,
		},
		{
			managerNodeID: manager2,
			limit:         5,
			want: []*Account{
				{ID: acc2, Label: "account-2", Index: []uint32{0, 0}},
			},
			wantLast: acc2,
		},
		{
			managerNodeID: "nonexistent",
			want:          nil,
		},
		{
			managerNodeID: manager1,
			limit:         2,
			want: []*Account{
				{ID: acc3, Label: "account-3", Index: []uint32{0, 2}},
				{ID: acc1, Label: "account-1", Index: []uint32{0, 1}},
			},
			wantLast: acc1,
		},
		{
			managerNodeID: manager1,
			limit:         2,
			prev:          acc1,
			want: []*Account{
				{ID: acc0, Label: "account-0", Index: []uint32{0, 0}},
			},
			wantLast: acc0,
		},
		{
			managerNodeID: manager1,
			limit:         2,
			prev:          acc0,
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
}

func TestGetAccount(t *testing.T) {
	ctx := pgtest.NewContext(t)

	acc0 := assettest.CreateAccountFixture(ctx, t, "", "account-0", nil)
	examples := []struct {
		id      string
		want    *Account
		wantErr error
	}{
		{
			acc0,
			&Account{ID: acc0, Label: "account-0", Index: []uint32{0, 0}},
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
}

func TestUpdateAccount(t *testing.T) {
	ctx := pgtest.NewContext(t)
	managerNode := newTestManagerNode(t, ctx, nil, "foo")
	account, err := CreateAccount(ctx, managerNode.ID, "foo", nil, nil)
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
}

// Test that calling UpdateManagerNode with no new label is a no-op.
func TestUpdateAccountNoUpdate(t *testing.T) {
	ctx := pgtest.NewContext(t)
	managerNode := newTestManagerNode(t, ctx, nil, "foo")
	account, err := CreateAccount(ctx, managerNode.ID, "foo", nil, nil)
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
}

func TestArchiveAccount(t *testing.T) {
	ctx := pgtest.NewContext(t)
	account := newTestAccount(t, ctx, nil, "account-1")
	err := ArchiveAccount(ctx, account.ID)
	if err != nil {
		t.Errorf("could not archive account with id %s: %v", account.ID, err)
	}

	var archived bool
	checkQ := `SELECT archived FROM accounts WHERE id = $1`
	err = pg.QueryRow(ctx, checkQ, account.ID).Scan(&archived)

	if !archived {
		t.Errorf("expected account %s to be archived", account.ID)
	}
}
