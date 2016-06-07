package appdb_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	. "chain/core/appdb"
	"chain/core/asset/assettest"
	"chain/core/generator"
	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/testutil"
)

func TestCreateAccount(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	managerNode := newTestManagerNode(t, ctx, nil, "foo")
	_, err := CreateAccount(ctx, managerNode.ID, "", nil, nil)
	if err == nil {
		t.Error("err = nil, want error")
	}
}

func TestCreateAccountWithKey(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
	keys := []string{"xpub6AqXYTtDPZ5NYt1xwRWRojirrYTHxyGnv3HHzeXTuJdAznKWVtEhj7sVzyMuJMn1E65uhw7pozjFsFaa4nRJBiDijr7do4zZ1CwM8TjTP3G"}
	_, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", keys, nil)
	if err != nil {
		t.Error("unexpected error", err)
	}
}

func TestCreateAccountWithMissingKey(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
	_, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", nil, nil)
	if err == nil {
		t.Error("err = nil, want error")
	}
}

func TestCreateAccountWithInvalidKey(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
	keys := []string{"keyo"}
	_, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", keys, nil)
	if err == nil {
		t.Error("err = nil, want error")
	}
}

func TestCreateAccountWithTooManyKeys(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
	keys := []string{"keyo", "keya", "keyeeeee"}
	_, err := CreateAccount(ctx, managerNode.ID, "varfootooyoutoo", keys, nil)
	if err == nil {
		t.Error("err = nil, want error")
	}
}

func TestCreateAccountIdempotency(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	managerNode := newTestVarKeyManagerNode(t, ctx, nil, "varfoo", 1, 1)
	keys := []string{"xpub6AqXYTtDPZ5NYt1xwRWRojirrYTHxyGnv3HHzeXTuJdAznKWVtEhj7sVzyMuJMn1E65uhw7pozjFsFaa4nRJBiDijr7do4zZ1CwM8TjTP3G"}

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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	account := newTestAccount(t, ctx, nil, "account-1")
	err := ArchiveAccount(ctx, account.ID)
	if err != nil {
		t.Errorf("could not archive account with id %s: %v", account.ID, err)
	}

	var archived bool
	checkQ := `SELECT archived FROM accounts WHERE id = $1`
	err = pg.QueryRow(ctx, checkQ, account.ID).Scan(&archived)
	if err != nil {
		t.Error(err)
	}
	if !archived {
		t.Errorf("expected account %s to be archived", account.ID)
	}
}

func TestListAccountUTXOs(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	_, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	asset0 := assettest.CreateAssetFixture(ctx, t, "", "", "")
	asset1 := assettest.CreateAssetFixture(ctx, t, "", "", "")
	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "manager-node-0", nil, nil)
	acc0 := assettest.CreateAccountFixture(ctx, t, mn0, "account-0", nil)
	acc1 := assettest.CreateAccountFixture(ctx, t, mn0, "account-1", nil)

	var issuances []state.Output
	issuances = append(issuances, assettest.IssueAssetsFixture(ctx, t, asset0, 1, acc0))
	issuances = append(issuances, assettest.IssueAssetsFixture(ctx, t, asset0, 2, acc1))
	issuances = append(issuances, assettest.IssueAssetsFixture(ctx, t, asset1, 3, acc0))
	issuances = append(issuances, assettest.IssueAssetsFixture(ctx, t, asset1, 4, acc1))

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	var (
		wantTxOuts []*TxOutput
		next       []string
	)
	for i, iss := range issuances {
		if iss.Metadata == nil {
			iss.Metadata = []byte{}
		}

		wantTxOuts = append(wantTxOuts, &TxOutput{
			TxHash:   iss.Outpoint.Hash,
			TxIndex:  iss.Outpoint.Index,
			AssetID:  iss.AssetID,
			Amount:   iss.Amount,
			Script:   iss.Script,
			Address:  iss.Script,
			Metadata: iss.Metadata,
		})

		next = append(next, fmt.Sprintf("2-%d-%d", i, iss.Outpoint.Index))
	}

	examples := []struct {
		accountID string
		assetIDs  []bc.AssetID
		prev      string
		limit     int

		want     []*TxOutput
		wantNext string
		wantErr  error
	}{
		// acc0, no asset ID filter
		{
			acc0,
			nil,
			"",
			50,

			[]*TxOutput{wantTxOuts[0], wantTxOuts[2]},
			"2-2-0",
			nil,
		},
		// acc1, no asset ID filter
		{
			acc1,
			nil,
			"",
			50,

			[]*TxOutput{wantTxOuts[1], wantTxOuts[3]},
			"2-3-0",
			nil,
		},
		// acc0, filter by existing asset
		{
			acc0,
			[]bc.AssetID{asset0},
			"",
			50,

			[]*TxOutput{wantTxOuts[0]},
			"2-0-0",
			nil,
		},
		// acc0, filter by unrecognized asset
		{
			acc0,
			[]bc.AssetID{bc.AssetID{}},
			"",
			50,

			nil,
			"",
			nil,
		},
		// acc0, pagination 1/3
		{
			acc0,
			nil,
			"",
			1,

			[]*TxOutput{wantTxOuts[0]},
			"2-0-0",
			nil,
		},
		// acc0, pagination 2/3
		{
			acc0,
			nil,
			"2-0-0",
			1,

			[]*TxOutput{wantTxOuts[2]},
			"2-2-0",
			nil,
		},
		// acc0, pagination 3/3
		{
			acc0,
			nil,
			"2-2-0",
			1,

			nil,
			"",
			nil,
		},
	}

	for i, ex := range examples {
		t.Logf("Example %d", i+1)

		got, gotNext, gotErr := ListAccountUTXOs(ctx, ex.accountID, ex.assetIDs, ex.prev, ex.limit)

		if !reflect.DeepEqual(got, ex.want) {
			g, _ := json.Marshal(got)
			w, _ := json.Marshal(ex.want)
			t.Errorf("results:\ngot:  %s\nwant: %s", g, w)
		}

		if gotNext != ex.wantNext {
			t.Errorf("next cursor:\ngot:  %v\nwant: %v", gotNext, ex.wantNext)
		}

		if gotErr != errors.Root(ex.wantErr) {
			t.Errorf("error:\ngot:  %v\nwant: %v", gotErr, ex.wantErr)
		}
	}
}
