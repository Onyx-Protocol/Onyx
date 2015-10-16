package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
)

func TestInsertWallet(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleProjectFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	wallet, err := InsertWallet(ctx, "proj-id-0", "foo", []*hdkey.XKey{dummyXPub}, nil)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if wallet.ID == "" {
		t.Errorf("got empty wallet id")
	}
}

func TestGetWallet(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0');

		INSERT INTO manager_nodes (id, project_id, key_index, label) VALUES
			('wallet-id-0', 'proj-id-0', 0, 'wallet-0');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		id      string
		want    *Wallet
		wantErr error
	}{
		{
			"wallet-id-0",
			&Wallet{ID: "wallet-id-0", Label: "wallet-0", Blockchain: "sandbox"},
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

		got, gotErr := GetWallet(ctx, ex.id)

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("wallet:\ngot:  %v\nwant: %v", got, ex.want)
		}

		if errors.Root(gotErr) != ex.wantErr {
			t.Errorf("get wallet error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
		}
	}
}

func TestWalletBalance(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO utxos (txid, index, asset_id, amount, address_id, account_id, manager_node_id)
		VALUES ('t0', 0, 'a1', 10, 'add0', 'b0', 'w1'),
		       ('t1', 1, 'a1', 5, 'add0', 'b0', 'w1'),
		       ('t2', 2, 'a2', 20, 'add0', 'b1', 'w1');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	cases := []struct {
		wID      string
		prev     string
		limit    int
		want     []*Balance
		wantLast string
	}{{
		wID:      "w1",
		limit:    5,
		want:     []*Balance{{"a1", 15, 15}, {"a2", 20, 20}},
		wantLast: "a2",
	}, {
		wID:      "w1",
		prev:     "a1",
		limit:    5,
		want:     []*Balance{{"a2", 20, 20}},
		wantLast: "a2",
	}, {
		wID:      "w1",
		prev:     "a2",
		limit:    5,
		want:     nil,
		wantLast: "",
	}, {
		wID:      "w1",
		limit:    1,
		want:     []*Balance{{"a1", 15, 15}},
		wantLast: "a1",
	}, {
		wID:      "nonexistent",
		limit:    5,
		want:     nil,
		wantLast: "",
	}}

	for _, c := range cases {
		got, gotLast, err := WalletBalance(ctx, c.wID, c.prev, c.limit)
		if err != nil {
			t.Errorf("WalletBalance(%s, %s, %d): unexpected error %v", c.wID, c.prev, c.limit, err)
			continue
		}

		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("WalletBalance(%s, %s, %d) = %v want %v", c.wID, c.prev, c.limit, got, c.want)
		}

		if gotLast != c.wantLast {
			t.Errorf("WalletBalance(%s, %s, %d) = %v want %v", c.wID, c.prev, c.limit, gotLast, c.wantLast)
		}
	}
}

func TestListWallets(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0'),
			('app-id-1', 'app-1');

		INSERT INTO manager_nodes (id, project_id, key_index, label, created_at) VALUES
			-- insert in reverse chronological order, to ensure that ListWallets
			-- is performing a sort.
			('wallet-id-0', 'proj-id-0', 0, 'wallet-0', now()),
			('wallet-id-1', 'proj-id-0', 1, 'wallet-1', now() - '1 day'::interval),

			('wallet-id-2', 'app-id-1', 2, 'wallet-2', now());
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		projID string
		want   []*Wallet
	}{
		{
			"proj-id-0",
			[]*Wallet{
				{ID: "wallet-id-1", Blockchain: "sandbox", Label: "wallet-1"},
				{ID: "wallet-id-0", Blockchain: "sandbox", Label: "wallet-0"},
			},
		},
		{
			"app-id-1",
			[]*Wallet{
				{ID: "wallet-id-2", Blockchain: "sandbox", Label: "wallet-2"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.projID)

		got, err := ListWallets(ctx, ex.projID)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("wallets:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}
