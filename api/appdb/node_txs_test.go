package appdb_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	. "chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

const sampleTxFixture = `
	INSERT INTO manager_nodes (id, project_id, label, current_rotation, key_index)
		VALUES('mn0', 'proj-id-0', '', 'c0', 0);
	INSERT INTO manager_txs (id, manager_node_id, data, txid)
		VALUES('mtx0', 'mn0', '{"outputs":"boop"}', 'tx0');
`

func TestWriteManagerTx(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		accounts := []string{"account-1", "account-2"}
		err := WriteManagerTx(ctx, "tx1", []byte(`{}`), "mnode-1", accounts)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		txs, _, err := ManagerTxs(ctx, "mnode-1", "", 100)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		if len(txs) != 1 {
			t.Fatal("expected manager tx to be in db")
		}

		for _, acc := range accounts {
			txs, _, err = AccountTxs(ctx, acc, "", 100)
			if err != nil {
				t.Log(errors.Stack(err))
				t.Fatal(err)
			}

			if len(txs) != 1 {
				t.Errorf("expected account tx to be in db for %s", acc)
			}
		}
	})
}

func TestWriteIssuerTx(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		err := WriteIssuerTx(ctx, "tx1", []byte(`{}`), "inode-1", "asset-1")
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		txs, _, err := IssuerTxs(ctx, "inode-1", "", 100)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		if len(txs) != 1 {
			t.Fatal("expected issuer tx to be in db")
		}

		txs, _, err = AssetTxs(ctx, "asset-1", "", 100)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		if len(txs) != 1 {
			t.Fatal("expected asset tx to be in db")
		}
	})
}

func TestManagerTxs(t *testing.T) {
	ctx := pgtest.NewContext(t, sampleProjectFixture, sampleTxFixture)
	defer pgtest.Finish(ctx)

	txs, last, err := ManagerTxs(ctx, "mn0", "mtx2", 1) // mtx2 would be a newer item than mtx1
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(txs) != 1 {
		t.Fatalf("want len(txs)=1 got=%d", len(txs))
	}

	if last != "mtx0" {
		t.Fatalf("want last txs to be mtx0 got=%v", last)
	}

	if string(*txs[0]) != `{"outputs":"boop"}` {
		t.Fatalf("want={outputs: boop}, got=%v", *txs[0])
	}
}

func TestManagerTxsLimit(t *testing.T) {
	ctx := pgtest.NewContext(t, sampleProjectFixture, sampleTxFixture, `
		INSERT INTO manager_txs (id, manager_node_id, data, txid)
			VALUES
				('mtx1', 'mn0', '{"outputs":"coop"}', 'tx1'),
				('mtx2', 'mn0', '{"outputs":"doop"}', 'tx2'),
				('mtx3', 'mn0', '{"outputs":"foop"}', 'tx3');
	`)
	defer pgtest.Finish(ctx)

	txs, last, err := ManagerTxs(ctx, "mn0", "mtx4", 2) // mtx4 would be a newer item than mtx1
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(txs) != 2 {
		t.Log(txs)
		t.Fatalf("want len(txs)=2 got=%d", len(txs))
	}

	if last != "mtx2" {
		t.Fatalf("want last txs to be mtx2 got=%v", last)
	}

	if string(*txs[0]) != `{"outputs":"foop"}` {
		t.Fatalf("want={outputs: foop}, got=%v", *txs[0])
	}

	if string(*txs[1]) != `{"outputs":"doop"}` {
		t.Fatalf("want={outputs: doop}, got=%v", *txs[1])
	}
}

func TestAccountTxs(t *testing.T) {
	ctx := pgtest.NewContext(t, sampleProjectFixture, sampleTxFixture, `
		INSERT INTO accounts (id, manager_node_id, key_index) VALUES('acc0', 'mn0', 0);
		INSERT INTO manager_txs_accounts VALUES ('mtx0', 'acc0');
	`)

	defer pgtest.Finish(ctx)

	txs, last, err := AccountTxs(ctx, "acc0", "mtx1", 1)
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(txs) != 1 {
		t.Fatalf("want len(txs)=1 got=%d", len(txs))
	}

	if last != "mtx0" {
		t.Fatalf("want last txs to be mtx0 got=%v", last)
	}
}

func TestAccountTxsLimit(t *testing.T) {
	ctx := pgtest.NewContext(t, sampleProjectFixture, sampleTxFixture, `
		INSERT INTO manager_txs (id, manager_node_id, data, txid)
			VALUES
			('mtx1', 'mn0', '{"outputs":"coop"}', 'tx1'),
			('mtx2', 'mn0', '{"outputs":"doop"}', 'tx2'),
			('mtx3', 'mn0', '{"outputs":"foop"}', 'tx3');
		INSERT INTO accounts (id, manager_node_id, key_index) VALUES('acc0', 'mn0', 0);
		INSERT INTO manager_txs_accounts VALUES
			('mtx0', 'acc0'),
			('mtx1', 'acc0'),
			('mtx2', 'acc0'),
			('mtx3', 'acc0');
	`)
	defer pgtest.Finish(ctx)

	txs, last, err := AccountTxs(ctx, "acc0", "mtx4", 2) // mtx4 would be a newer item than mtx1
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(txs) != 2 {
		t.Log(txs)
		t.Fatalf("want len(txs)=2 got=%d", len(txs))
	}

	if last != "mtx2" {
		t.Fatalf("want last txs to be mtx2 got=%v", last)
	}

	if string(*txs[0]) != `{"outputs":"foop"}` {
		t.Fatalf("want={outputs: foop}, got=%v", *txs[0])
	}

	if string(*txs[1]) != `{"outputs":"doop"}` {
		t.Fatalf("want={outputs: doop}, got=%v", *txs[1])
	}
}

func TestIssuerTxs(t *testing.T) {
	ctx := pgtest.NewContext(t, writeActivityFix, `
		INSERT INTO issuer_txs
			(id, issuer_node_id, data, txid)
		VALUES
			('itx-id-0', 'in-id-0', '{"transaction_id": "tx-id-0"}', 'tx-id-0'),
			('itx-id-1', 'in-id-1', '{"transaction_id": "tx-id-1"}', 'tx-id-1'),
			('itx-id-2', 'in-id-0', '{"transaction_id": "tx-id-2"}', 'tx-id-2');
	`)
	defer pgtest.Finish(ctx)

	examples := []struct {
		inodeID  string
		wantAct  []*json.RawMessage
		wantLast string
	}{
		{
			"in-id-0",
			stringsToRawJSON(
				`{"transaction_id": "tx-id-2"}`,
				`{"transaction_id": "tx-id-0"}`,
			),
			"itx-id-0",
		},
		{
			"in-id-1",
			stringsToRawJSON(
				`{"transaction_id": "tx-id-1"}`,
			),
			"itx-id-1",
		},
	}

	for _, ex := range examples {
		t.Log("Example", ex.inodeID)

		gotAct, gotLast, err := IssuerTxs(ctx, ex.inodeID, "", 50)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(gotAct, ex.wantAct) {
			t.Errorf("txs:\ngot:  %v\nwant: %v", gotAct, ex.wantAct)
		}

		if gotLast != ex.wantLast {
			t.Errorf("last got = %v want %v", gotLast, ex.wantLast)
		}
	}
}

func TestAssetTxs(t *testing.T) {
	ctx := pgtest.NewContext(t, writeActivityFix, `
		INSERT INTO issuer_txs
			(id, issuer_node_id, data, txid)
		VALUES
			('itx-id-0', 'in-id-0', '{"transaction_id": "tx-id-0"}', 'tx-id-0'),
			('itx-id-1', 'in-id-1', '{"transaction_id": "tx-id-1"}', 'tx-id-1'),
			('itx-id-2', 'in-id-0', '{"transaction_id": "tx-id-2"}', 'tx-id-2');

		INSERT INTO issuer_txs_assets
			(issuer_tx_id, asset_id)
		VALUES
			('itx-id-0', 'asset-id-0'),
			('itx-id-1', 'asset-id-1'),
			('itx-id-2', 'asset-id-0');
	`)
	defer pgtest.Finish(ctx)

	stringsToRawJSON := func(strs ...string) []*json.RawMessage {
		var res []*json.RawMessage
		for _, s := range strs {
			b := json.RawMessage([]byte(s))
			res = append(res, &b)
		}
		return res
	}

	examples := []struct {
		assetID  string
		wantAct  []*json.RawMessage
		wantLast string
	}{
		{
			"asset-id-0",
			stringsToRawJSON(`{"transaction_id": "tx-id-2"}`, `{"transaction_id": "tx-id-0"}`),
			"itx-id-0",
		},
		{
			"asset-id-1",
			stringsToRawJSON(`{"transaction_id": "tx-id-1"}`),
			"itx-id-1",
		},
	}

	for _, ex := range examples {
		t.Log("Example", ex.assetID)

		gotAct, gotLast, err := AssetTxs(ctx, ex.assetID, "", 50)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(gotAct, ex.wantAct) {
			t.Errorf("txs:\ngot:  %v\nwant: %v", gotAct, ex.wantAct)
		}

		if gotLast != ex.wantLast {
			t.Errorf("last got = %v want %v", gotLast, ex.wantLast)
		}
	}
}

func TestManagerTx(t *testing.T) {
	ctx := pgtest.NewContext(t, sampleProjectFixture, sampleTxFixture)
	defer pgtest.Finish(ctx)

	txs, err := ManagerTx(ctx, "mn0", "tx0")
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if string(*txs) != `{"outputs":"boop"}` {
		t.Fatalf("want={outputs: boop}, got=%s", *txs)
	}

	_, err = ManagerTx(ctx, "mn0", "txDoesNotExist")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Fatalf("want=%v got=%v", pg.ErrUserInputNotFound, err)
	}
}
