package appdb

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/api/utxodb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
)

// Addresses formerly in fixture.
// These will now be sent to and echoed from the client,
// no longer stored in the db.
//   (id, manager_node_id, account_id, keyset, key_index, address, redeem_script, pk_script, is_change)
//   ('addr-id-0', 'manager-node-id-0', 'account-id-0', '{}', 0, 'addr-0', '{}', '{}', false),
//   ('addr-id-1', 'manager-node-id-0', 'account-id-0', '{}', 1, 'addr-1', '{}', '{}', true),
//   ('addr-id-2', 'manager-node-id-0', 'account-id-1', '{}', 0, 'addr-2', '{}', '{}', false),
//   ('addr-id-3', 'manager-node-id-0', 'account-id-1', '{}', 1, 'addr-3', '{}', '{}', true),
//   ('addr-id-4', 'manager-node-id-1', 'account-id-2', '{}', 0, 'addr-4', '{}', '{}', false),
//   ('addr-id-5', 'manager-node-id-1', 'account-id-2', '{}', 1, 'addr-5', '{}', '{}', true);

const writeActivityFix = `
	INSERT INTO projects
		(id, name)
	VALUES
		('proj-id-0', 'proj-0');

	INSERT INTO manager_nodes
		(id, project_id, key_index, label, current_rotation, sigs_required)
	VALUES
		('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0', 'rot-id-0', 1),
		('manager-node-id-1', 'proj-id-0', 0, 'manager-node-1', 'rot-id-1', 1);

	INSERT INTO accounts
		(id, manager_node_id, key_index, label)
	VALUES
		('account-id-0', 'manager-node-id-0', 0, 'account-0'),
		('account-id-1', 'manager-node-id-0', 1, 'account-1'),
		('account-id-2', 'manager-node-id-1', 0, 'account-2');

	INSERT INTO issuer_nodes
		(id, project_id, key_index, label, keyset)
	VALUES
		('in-id-0', 'proj-id-0', 0, 'in-0', '{}'),
		('in-id-1', 'proj-id-0', 1, 'in-1', '{}');

	INSERT INTO assets
		(id, issuer_node_id, key_index, redeem_script, label)
	VALUES
		('asset-id-0', 'in-id-0', 0, '{}', 'asset-0'),
		('asset-id-1', 'in-id-0', 1, '{}', 'asset-1'),
		('asset-id-2', 'in-id-1', 0, '{}', 'asset-2');

	INSERT INTO rotations
		(id, manager_node_id, keyset)
	VALUES
		('rot-id-0', 'manager-node-id-0', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}'),
		('rot-id-1', 'manager-node-id-1', '{xpub661MyMwAqRbcGiDB8FQvHnDAZyaGUyzm3qN1Q3NDJz1PgAWCfyi9WRCS7Z9HyM5QNEh45fMyoaBMqjfoWPdnktcN8chJYB57D2Y7QtNmadr}');
`

const sampleActivityFixture = `
	INSERT INTO manager_nodes (id, project_id, label, current_rotation, key_index)
		VALUES('mn0', 'proj-id-0', '', 'c0', 0);
	INSERT INTO activity (id, manager_node_id, data, txid)
		VALUES('act0', 'mn0', '{"outputs":"boop"}', 'tx0');
`

func TestManagerNodeActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleProjectFixture, sampleActivityFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	activity, last, err := ManagerNodeActivity(ctx, "mn0", "act2", 1) // act2 would be a newer item than act1
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(activity) != 1 {
		t.Fatalf("want len(activity)=1 got=%d", len(activity))
	}

	if last != "act0" {
		t.Fatalf("want last activity to be act0 got=%v", last)
	}

	if string(*activity[0]) != `{"outputs":"boop"}` {
		t.Fatalf("want={outputs: boop}, got=%v", *activity[0])
	}
}

func TestManagerNodeActivityLimit(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleProjectFixture, sampleActivityFixture, `
		INSERT INTO activity (id, manager_node_id, data, txid)
			VALUES
				('act1', 'mn0', '{"outputs":"coop"}', 'tx1'),
				('act2', 'mn0', '{"outputs":"doop"}', 'tx2'),
				('act3', 'mn0', '{"outputs":"foop"}', 'tx3');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	activity, last, err := ManagerNodeActivity(ctx, "mn0", "act4", 2) // act4 would be a newer item than act1
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(activity) != 2 {
		t.Log(activity)
		t.Fatalf("want len(activity)=2 got=%d", len(activity))
	}

	if last != "act2" {
		t.Fatalf("want last activity to be act2 got=%v", last)
	}

	if string(*activity[0]) != `{"outputs":"foop"}` {
		t.Fatalf("want={outputs: foop}, got=%v", *activity[0])
	}

	if string(*activity[1]) != `{"outputs":"doop"}` {
		t.Fatalf("want={outputs: doop}, got=%v", *activity[1])
	}
}

func TestAccountActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleProjectFixture, sampleActivityFixture, `
		INSERT INTO accounts (id, manager_node_id, key_index) VALUES('acc0', 'mn0', 0);
		INSERT INTO activity_accounts VALUES ('act0', 'acc0');
	`)

	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	activity, last, err := AccountActivity(ctx, "acc0", "act1", 1)
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(activity) != 1 {
		t.Fatalf("want len(activity)=1 got=%d", len(activity))
	}

	if last != "act0" {
		t.Fatalf("want last activity to be act0 got=%v", last)
	}
}

func TestAccountActivityLimit(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleProjectFixture, sampleActivityFixture, `
		INSERT INTO activity (id, manager_node_id, data, txid)
			VALUES
			('act1', 'mn0', '{"outputs":"coop"}', 'tx1'),
			('act2', 'mn0', '{"outputs":"doop"}', 'tx2'),
			('act3', 'mn0', '{"outputs":"foop"}', 'tx3');
		INSERT INTO accounts (id, manager_node_id, key_index) VALUES('acc0', 'mn0', 0);
		INSERT INTO activity_accounts VALUES
			('act0', 'acc0'),
			('act1', 'acc0'),
			('act2', 'acc0'),
			('act3', 'acc0');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	activity, last, err := AccountActivity(ctx, "acc0", "act4", 2) // act4 would be a newer item than act1
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(activity) != 2 {
		t.Log(activity)
		t.Fatalf("want len(activity)=2 got=%d", len(activity))
	}

	if last != "act2" {
		t.Fatalf("want last activity to be act2 got=%v", last)
	}

	if string(*activity[0]) != `{"outputs":"foop"}` {
		t.Fatalf("want={outputs: foop}, got=%v", *activity[0])
	}

	if string(*activity[1]) != `{"outputs":"doop"}` {
		t.Fatalf("want={outputs: doop}, got=%v", *activity[1])
	}
}

func TestIssuerNodeActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix, `
		INSERT INTO issuance_activity
			(id, issuer_node_id, data, txid)
		VALUES
			('ia-id-0', 'in-id-0', '{"transaction_id": "tx-id-0"}', 'tx-id-0'),
			('ia-id-1', 'in-id-1', '{"transaction_id": "tx-id-1"}', 'tx-id-1'),
			('ia-id-2', 'in-id-0', '{"transaction_id": "tx-id-2"}', 'tx-id-2');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

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
			"ia-id-0",
		},
		{
			"in-id-1",
			stringsToRawJSON(
				`{"transaction_id": "tx-id-1"}`,
			),
			"ia-id-1",
		},
	}

	for _, ex := range examples {
		t.Log("Example", ex.inodeID)

		gotAct, gotLast, err := IssuerNodeActivity(ctx, ex.inodeID, "", 50)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(gotAct, ex.wantAct) {
			t.Errorf("activity:\ngot:  %v\nwant: %v", gotAct, ex.wantAct)
		}

		if gotLast != ex.wantLast {
			t.Errorf("last got = %v want %v", gotLast, ex.wantLast)
		}
	}
}

func TestAssetActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix, `
		INSERT INTO issuance_activity
			(id, issuer_node_id, data, txid)
		VALUES
			('ia-id-0', 'in-id-0', '{"transaction_id": "tx-id-0"}', 'tx-id-0'),
			('ia-id-1', 'in-id-1', '{"transaction_id": "tx-id-1"}', 'tx-id-1'),
			('ia-id-2', 'in-id-0', '{"transaction_id": "tx-id-2"}', 'tx-id-2');

		INSERT INTO issuance_activity_assets
			(issuance_activity_id, asset_id)
		VALUES
			('ia-id-0', 'asset-id-0'),
			('ia-id-1', 'asset-id-1'),
			('ia-id-2', 'asset-id-0');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

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
			"ia-id-0",
		},
		{
			"asset-id-1",
			stringsToRawJSON(`{"transaction_id": "tx-id-1"}`),
			"ia-id-1",
		},
	}

	for _, ex := range examples {
		t.Log("Example", ex.assetID)

		gotAct, gotLast, err := AssetActivity(ctx, ex.assetID, "", 50)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(gotAct, ex.wantAct) {
			t.Errorf("activity:\ngot:  %v\nwant: %v", gotAct, ex.wantAct)
		}

		if gotLast != ex.wantLast {
			t.Errorf("last got = %v want %v", gotLast, ex.wantLast)
		}
	}
}

func TestActivityByTxID(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleProjectFixture, sampleActivityFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	activity, err := ManagerNodeTxActivity(ctx, "mn0", "tx0")
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if string(*activity) != `{"outputs":"boop"}` {
		t.Fatalf("want={outputs: boop}, got=%s", *activity)
	}

	_, err = ManagerNodeTxActivity(ctx, "mn0", "txDoesNotExist")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Fatalf("want=%v got=%v", pg.ErrUserInputNotFound, err)
	}
}

// TestWriteActivity is an integration test for WriteActivity. Only basic use
// cases are covered here. Edge cases are covered in unit tests that follow.
func TestWriteActivity(t *testing.T) {
	// Mock transactions for creating prevouts.
	prevTxA := &bc.Tx{Version: bc.CurrentTransactionVersion}
	prevTxA.Outputs = append(prevTxA.Outputs, &bc.TxOutput{AssetID: bc.AssetID{}, Value: 123})
	prevTxB := &bc.Tx{Version: bc.CurrentTransactionVersion}
	prevTxB.Outputs = append(prevTxB.Outputs, &bc.TxOutput{AssetID: bc.AssetID([32]byte{1}), Value: 456})
	txTime := time.Now().UTC()

	examples := []struct {
		tx                      *bc.Tx
		outs                    []*UTXO
		fixture                 string
		wantManagerNodeActivity map[string]actItem
		wantAccounts            []string
		wantIssuanceActivity    map[string]actItem
		wantAssets              []string
	}{
		// Issuance
		{
			tx: &bc.Tx{
				// Issuances have a single prevout with an empty tx hash.
				Inputs: []*bc.TxInput{{Previous: bc.IssuanceOutpoint}},
				// The content of the outs is irrelevant for the test.
				// Issuances require at least one output.
				Outputs: []*bc.TxOutput{{}},
			},
			outs: []*UTXO{
				{
					UTXO: &utxodb.UTXO{
						AccountID: "account-id-0",
						AssetID:   "asset-id-0",
						Amount:    1,
						Outpoint:  bc.Outpoint{Hash: mustHashFromStr("db49adbf4b456581d39b610b2e422e21807086c108d01c33363c2c488dc02b12"), Index: 0},
					},
					Addr:          "addr-0",
					ManagerNodeID: "manager-node-id-0",
					IsChange:      false,
				},
				{
					UTXO: &utxodb.UTXO{
						AccountID: "account-id-2",
						AssetID:   "asset-id-0",
						Amount:    2,
						Outpoint:  bc.Outpoint{Hash: mustHashFromStr("db49adbf4b456581d39b610b2e422e21807086c108d01c33363c2c488dc02b12"), Index: 0},
					},
					Addr:          "addr-4",
					ManagerNodeID: "manager-node-id-1",
					IsChange:      false,
				},
			},
			wantManagerNodeActivity: map[string]actItem{
				"manager-node-id-0": actItem{
					TxHash: "9cb6150ade7117fd27bed7ae03ee54716afdde976a654e932baacea225f65b9e",
					Time:   txTime,
					Inputs: []actEntry{},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, AccountID: "account-id-0", AccountLabel: "account-0"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, Address: "addr-4"},
					},
				},
				"manager-node-id-1": actItem{
					TxHash: "9cb6150ade7117fd27bed7ae03ee54716afdde976a654e932baacea225f65b9e",
					Time:   txTime,
					Inputs: []actEntry{},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, AccountID: "account-id-2", AccountLabel: "account-2"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, Address: "addr-0"},
					},
				},
			},
			wantAccounts: []string{"account-id-0", "account-id-2"},
			wantIssuanceActivity: map[string]actItem{
				"in-id-0": actItem{
					TxHash: "9cb6150ade7117fd27bed7ae03ee54716afdde976a654e932baacea225f65b9e",
					Time:   txTime,
					Inputs: []actEntry{},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, AccountID: "account-id-0", AccountLabel: "account-0"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, AccountID: "account-id-2", AccountLabel: "account-2"},
					},
				},
			},
			wantAssets: []string{"asset-id-0"},
		},

		// Transfer with change
		{
			tx: &bc.Tx{
				Inputs: []*bc.TxInput{{
					Previous: bc.Outpoint{Hash: mustHashFromStr("4786c29077265138e00a8fce822c5fb998c0ce99df53d939bb53d81bca5aa426"), Index: 0},
				}},
			},
			outs: []*UTXO{
				{
					UTXO: &utxodb.UTXO{
						AccountID: "account-id-2",
						AssetID:   "asset-id-0",
						Amount:    1,
						Outpoint:  bc.Outpoint{Hash: mustHashFromStr("db49adbf4b456581d39b610b2e422e21807086c108d01c33363c2c488dc02b12"), Index: 0},
					},
					Addr:          "addr-4",
					ManagerNodeID: "manager-node-id-1",
					IsChange:      false,
				},

				{
					UTXO: &utxodb.UTXO{
						AccountID: "account-id-0",
						AssetID:   "asset-id-0",
						Amount:    2,
						Outpoint:  bc.Outpoint{Hash: mustHashFromStr("db49adbf4b456581d39b610b2e422e21807086c108d01c33363c2c488dc02b12"), Index: 1},
					},
					Addr:          "addr-1",
					ManagerNodeID: "manager-node-id-0",
					IsChange:      true,
				},
			},
			fixture: `
				INSERT INTO utxos
					(txid, index, asset_id, amount, addr_index, account_id, manager_node_id)
				VALUES
					('4786c29077265138e00a8fce822c5fb998c0ce99df53d939bb53d81bca5aa426', 0, 'asset-id-0', 3, 0, 'account-id-0', 'manager-node-id-0');
			`,
			wantManagerNodeActivity: map[string]actItem{
				"manager-node-id-0": actItem{
					TxHash: "6750e5583a6ae5156d7522efa60332448cff475d7fc4dcecd20979540c13c392",
					Time:   txTime,
					Inputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, AccountID: "account-id-0", AccountLabel: "account-0"},
					},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, Address: "addr-4"},
					},
				},
				"manager-node-id-1": actItem{
					TxHash: "6750e5583a6ae5156d7522efa60332448cff475d7fc4dcecd20979540c13c392",
					Time:   txTime,
					Inputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 3, Address: "32g4QsxVQrhZeXyXTUnfSByNBAdTfVUdVK"},
					},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, AccountID: "account-id-2", AccountLabel: "account-2"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, Address: "addr-1"},
					},
				},
			},
			wantAccounts:         []string{"account-id-0", "account-id-2"},
			wantIssuanceActivity: make(map[string]actItem),
		},
	}

	for i, ex := range examples {
		t.Log("Example", i)

		func() {
			txHash := ex.tx.Hash().String()

			dbtx := pgtest.TxWithSQL(t, writeActivityFix, ex.fixture)
			ctx := pg.NewContext(context.Background(), dbtx)
			defer dbtx.Rollback()

			err := WriteActivity(ctx, ex.tx, ex.outs, txTime)
			if err != nil {
				t.Fatal("unexpected error:", withStack(err))
			}

			gotManagerNodeActivity, err := getTestActivity(ctx, txHash, false)
			if err != nil {
				t.Fatal("unexpected error", err)
			}

			if !reflect.DeepEqual(gotManagerNodeActivity, ex.wantManagerNodeActivity) {
				t.Errorf("manager node activity:\ngot:  %+v\nwant: %+v", gotManagerNodeActivity, ex.wantManagerNodeActivity)
			}

			gotAccounts, err := getTestActivityAccounts(ctx, txHash)
			if err != nil {
				t.Fatal("unexpected error", err)
			}

			if !reflect.DeepEqual(gotAccounts, ex.wantAccounts) {
				t.Errorf("accounts:\ngot:  %v\nwant: %v", gotAccounts, ex.wantAccounts)
			}

			gotIssuanceActivity, err := getTestActivity(ctx, txHash, true)
			if err != nil {
				t.Fatal("unexpected error", err)
			}

			if !reflect.DeepEqual(gotIssuanceActivity, ex.wantIssuanceActivity) {
				t.Errorf("issuance activity:\ngot:  %v\nwant: %v", gotIssuanceActivity, ex.wantIssuanceActivity)
			}

			gotAssets, err := getTestActivityAssets(ctx, txHash)
			if err != nil {
				t.Fatal("unexpected error", err)
			}

			if !reflect.DeepEqual(gotAssets, ex.wantAssets) {
				t.Errorf("assets:\ngot:  %v\nwant: %v", gotAssets, ex.wantAssets)
			}
		}()
	}
}

func TestGetActUTXOs(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix, `
		INSERT INTO utxos
			(txid, index, asset_id, amount, addr_index, account_id, manager_node_id)
		VALUES
			('0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 0, 'asset-id-0', 100, 0, 'account-id-0', 'manager-node-id-0'),
			('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 1, 'asset-id-0', 50, 1, 'account-id-1', 'manager-node-id-0'),
			('0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 1, 'asset-id-1', 25, 0, 'account-id-2', 'manager-node-id-1');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	got, err := getActUTXOs(
		ctx,
		[]string{
			"0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098",
			"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df",
		},
		[]uint32{
			0,
			1,
		},
	)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	want := []*UTXO{
		{
			UTXO:          &utxodb.UTXO{AccountID: "account-id-0", AssetID: "asset-id-0", Amount: 100},
			Addr:          "32g4QsxVQrhZeXyXTUnfSByNBAdTfVUdVK",
			ManagerNodeID: "manager-node-id-0",
		},
		{
			UTXO:          &utxodb.UTXO{AccountID: "account-id-1", AssetID: "asset-id-0", Amount: 50},
			Addr:          "34C2bE5U2vXG7Vbu8ZVDcQD5LZrV2nzxx5",
			ManagerNodeID: "manager-node-id-0",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fail()
		for i, u := range got {
			t.Logf("got %d: %+v %+v", i, u.UTXO, u)
		}
		for i, u := range want {
			t.Logf("want %d: %+v %+v", i, u.UTXO, u)
		}
	}
}

func TestGetIDsFromUTXOs(t *testing.T) {
	utxos := []*UTXO{
		{UTXO: &utxodb.UTXO{AccountID: "account-id-2", AssetID: "asset-id-2"}, ManagerNodeID: "manager-node-id-1"},
		{UTXO: &utxodb.UTXO{AccountID: "account-id-1", AssetID: "asset-id-2"}, ManagerNodeID: "manager-node-id-1"},
		{UTXO: &utxodb.UTXO{AccountID: "account-id-0", AssetID: "asset-id-2"}, ManagerNodeID: "manager-node-id-0"},
		{UTXO: &utxodb.UTXO{AccountID: "account-id-0", AssetID: "asset-id-1"}, ManagerNodeID: "manager-node-id-0"},
		{UTXO: &utxodb.UTXO{AccountID: "account-id-0", AssetID: "asset-id-0"}, ManagerNodeID: "manager-node-id-0"},
		{UTXO: &utxodb.UTXO{AccountID: "account-id-0", AssetID: "asset-id-0"}, ManagerNodeID: "manager-node-id-0"},
	}

	gAssetIDs, gAccountIDs, gManagerNodeIDs, gManagerNodeAccounts := getIDsFromUTXOs(utxos)

	wAssetIDs := []string{"asset-id-0", "asset-id-1", "asset-id-2"}
	wAccountIDs := []string{"account-id-0", "account-id-1", "account-id-2"}
	wManagerNodeIDs := []string{"manager-node-id-0", "manager-node-id-1"}
	wManagerNodeAccounts := map[string][]string{
		"manager-node-id-0": []string{"account-id-0"},
		"manager-node-id-1": []string{"account-id-1", "account-id-2"},
	}

	if !reflect.DeepEqual(gAssetIDs, wAssetIDs) {
		t.Errorf("assetIDs:\ngot:  %v\nwant: %v", gAssetIDs, wAssetIDs)
	}
	if !reflect.DeepEqual(gAccountIDs, wAccountIDs) {
		t.Errorf("assetIDs:\ngot:  %v\nwant: %v", gAccountIDs, wAccountIDs)
	}
	if !reflect.DeepEqual(gManagerNodeIDs, wManagerNodeIDs) {
		t.Errorf("assetIDs:\ngot:  %v\nwant: %v", gManagerNodeIDs, wManagerNodeIDs)
	}
	if !reflect.DeepEqual(gManagerNodeAccounts, wManagerNodeAccounts) {
		t.Errorf("assetIDs:\ngot:  %v\nwant: %v", gManagerNodeAccounts, wManagerNodeAccounts)
	}
}

func TestGetActAssets(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		assetIDs []string
		want     []*actAsset
	}{
		{
			[]string{"asset-id-0", "asset-id-2"},
			[]*actAsset{
				{id: "asset-id-0", label: "asset-0", inID: "in-id-0", projID: "proj-id-0"},
				{id: "asset-id-2", label: "asset-2", inID: "in-id-1", projID: "proj-id-0"},
			},
		},
		{
			[]string{"asset-id-1"},
			[]*actAsset{
				{id: "asset-id-1", label: "asset-1", inID: "in-id-0", projID: "proj-id-0"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.assetIDs)

		got, err := getActAssets(ctx, ex.assetIDs)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("assets:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestGetActAccounts(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		accountIDs []string
		want       []*actAccount
	}{
		{
			[]string{"account-id-0", "account-id-2"},
			[]*actAccount{
				{id: "account-id-0", label: "account-0", managerNodeID: "manager-node-id-0", projID: "proj-id-0"},
				{id: "account-id-2", label: "account-2", managerNodeID: "manager-node-id-1", projID: "proj-id-0"},
			},
		},
		{
			[]string{"account-id-1"},
			[]*actAccount{
				{id: "account-id-1", label: "account-1", managerNodeID: "manager-node-id-0", projID: "proj-id-0"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.accountIDs)

		got, err := getActAccounts(ctx, ex.accountIDs)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("accounts:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestCoalesceActivity(t *testing.T) {
	examples := []struct {
		ins, outs       []*UTXO
		visibleAccounts []string
		want            txRawActivity
	}{
		// Simple transfer from Alice's perspective
		{
			ins: []*UTXO{
				{UTXO: &utxodb.UTXO{AccountID: "alice-account-id-0", AssetID: "gold", Amount: 10}, Addr: "alice-addr-id-0"},
			},
			outs: []*UTXO{
				{UTXO: &utxodb.UTXO{AccountID: "bob-account-id-0", AssetID: "gold", Amount: 10}, Addr: "bob-addr-id-0"},
			},

			visibleAccounts: []string{"alice-account-id-0"},

			want: txRawActivity{
				insByAsset: map[string]map[string]int64{},
				insByAccount: map[string]map[string]int64{
					"alice-account-id-0": map[string]int64{"gold": 10},
				},
				outsByAsset: map[string]map[string]int64{
					"bob-addr-id-0": map[string]int64{"gold": 10},
				},
				outsByAccount: map[string]map[string]int64{},
			},
		},

		// Simple transfer from Bob's perspective
		{
			ins: []*UTXO{
				{UTXO: &utxodb.UTXO{AssetID: "gold", Amount: 10, AccountID: "alice-account-id-0"}, Addr: "alice-addr-id-0"},
			},
			outs: []*UTXO{
				{UTXO: &utxodb.UTXO{AssetID: "gold", Amount: 10, AccountID: "bob-account-id-0"}, Addr: "bob-addr-id-0"},
			},

			visibleAccounts: []string{"bob-account-id-0"},

			want: txRawActivity{
				insByAsset: map[string]map[string]int64{
					"alice-addr-id-0": map[string]int64{"gold": 10},
				},
				insByAccount: map[string]map[string]int64{},
				outsByAsset:  map[string]map[string]int64{},
				outsByAccount: map[string]map[string]int64{
					"bob-account-id-0": map[string]int64{"gold": 10},
				},
			},
		},

		// Trade from Alice's perspective
		{
			ins: []*UTXO{
				{UTXO: &utxodb.UTXO{AssetID: "gold", Amount: 20, AccountID: "alice-account-id-0"}, Addr: "alice-addr-id-0"},
				{UTXO: &utxodb.UTXO{AssetID: "silver", Amount: 10, AccountID: "bob-account-id-0"}, Addr: "bob-addr-id-0"},
				{UTXO: &utxodb.UTXO{AssetID: "silver", Amount: 10, AccountID: "bob-account-id-0"}, Addr: "bob-addr-id-1"},
			},
			outs: []*UTXO{
				{UTXO: &utxodb.UTXO{AssetID: "silver", Amount: 15, AccountID: "alice-account-id-0"}, Addr: "alice-addr-id-1"},
				{UTXO: &utxodb.UTXO{AssetID: "gold", Amount: 5, AccountID: "bob-account-id-0"}, Addr: "bob-addr-id-2"},

				{UTXO: &utxodb.UTXO{AssetID: "gold", Amount: 15, AccountID: "alice-account-id-0"}, Addr: "alice-addr-id-2", IsChange: true},
				{UTXO: &utxodb.UTXO{AssetID: "silver", Amount: 5, AccountID: "bob-account-id-0"}, Addr: "bob-addr-id-3", IsChange: true},
			},

			visibleAccounts: []string{"alice-account-id-0"},

			want: txRawActivity{
				insByAsset: map[string]map[string]int64{
					"bob-addr-id-0": map[string]int64{"silver": 10},
					"bob-addr-id-1": map[string]int64{"silver": 10},
				},
				insByAccount: map[string]map[string]int64{
					"alice-account-id-0": map[string]int64{"gold": 5},
				},
				outsByAsset: map[string]map[string]int64{
					"bob-addr-id-2": map[string]int64{"gold": 5},
					"bob-addr-id-3": map[string]int64{"silver": 5},
				},
				outsByAccount: map[string]map[string]int64{
					"alice-account-id-0": map[string]int64{"silver": 15},
				},
			},
		},

		// Trade from Bob's perspective
		{
			ins: []*UTXO{
				{UTXO: &utxodb.UTXO{AssetID: "gold", Amount: 20, AccountID: "alice-account-id-0"}, Addr: "alice-addr-id-0"},
				{UTXO: &utxodb.UTXO{AssetID: "silver", Amount: 10, AccountID: "bob-account-id-0"}, Addr: "bob-addr-id-0"},
				{UTXO: &utxodb.UTXO{AssetID: "silver", Amount: 10, AccountID: "bob-account-id-0"}, Addr: "bob-addr-id-1"},
			},
			outs: []*UTXO{
				{UTXO: &utxodb.UTXO{AssetID: "silver", Amount: 15, AccountID: "alice-account-id-0"}, Addr: "alice-addr-id-1"},
				{UTXO: &utxodb.UTXO{AssetID: "gold", Amount: 5, AccountID: "bob-account-id-0"}, Addr: "bob-addr-id-2"},

				{UTXO: &utxodb.UTXO{AssetID: "gold", Amount: 15, AccountID: "alice-account-id-0"}, Addr: "alice-addr-id-2", IsChange: true},
				{UTXO: &utxodb.UTXO{AssetID: "silver", Amount: 5, AccountID: "bob-account-id-0"}, Addr: "bob-addr-id-3", IsChange: true},
			},

			visibleAccounts: []string{"bob-account-id-0"},

			want: txRawActivity{
				insByAsset: map[string]map[string]int64{
					"alice-addr-id-0": map[string]int64{"gold": 20},
				},
				insByAccount: map[string]map[string]int64{
					"bob-account-id-0": map[string]int64{"silver": 15},
				},
				outsByAsset: map[string]map[string]int64{
					"alice-addr-id-1": map[string]int64{"silver": 15},
					"alice-addr-id-2": map[string]int64{"gold": 15},
				},
				outsByAccount: map[string]map[string]int64{
					"bob-account-id-0": map[string]int64{"gold": 5},
				},
			},
		},
	}

	for i, ex := range examples {
		t.Log("Example", i)

		got := coalesceActivity(ex.ins, ex.outs, ex.visibleAccounts)

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("coalesced activity:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestCreateActEntries(t *testing.T) {
	r := txRawActivity{
		insByAsset: map[string]map[string]int64{
			"space-mountain": map[string]int64{
				"asset-id-0": 1,
				"asset-id-1": 2,
			},
			"small-world": map[string]int64{
				"asset-id-2": 3,
				"asset-id-3": 4,
			},
		},
		insByAccount: map[string]map[string]int64{
			"account-id-0": map[string]int64{
				"asset-id-4": 5,
				"asset-id-5": 6,
			},
		},
		outsByAsset: map[string]map[string]int64{
			"teacups": map[string]int64{
				"asset-id-5": 6,
				"asset-id-4": 5,
			},
		},
		outsByAccount: map[string]map[string]int64{
			"account-id-1": map[string]int64{
				"asset-id-3": 4,
				"asset-id-2": 3,
			},
			"account-id-2": map[string]int64{
				"asset-id-1": 2,
				"asset-id-0": 1,
			},
		},
	}

	assetLabels := map[string]string{
		"asset-id-0": "gold",
		"asset-id-1": "silver",
		"asset-id-2": "frankincense",
		"asset-id-3": "myrrh",
		"asset-id-4": "avocado",
		"asset-id-5": "mango",
	}

	accountLabels := map[string]string{
		"account-id-0": "charlie",
		"account-id-1": "bob",
		"account-id-2": "alice",
	}

	gotIns, gotOuts := createActEntries(r, assetLabels, accountLabels)

	wantIns := []actEntry{
		actEntry{AccountID: "account-id-0", AccountLabel: "charlie", AssetID: "asset-id-4", AssetLabel: "avocado", Amount: 5},
		actEntry{AccountID: "account-id-0", AccountLabel: "charlie", AssetID: "asset-id-5", AssetLabel: "mango", Amount: 6},
		actEntry{Address: "small-world", AssetID: "asset-id-2", AssetLabel: "frankincense", Amount: 3},
		actEntry{Address: "small-world", AssetID: "asset-id-3", AssetLabel: "myrrh", Amount: 4},
		actEntry{Address: "space-mountain", AssetID: "asset-id-0", AssetLabel: "gold", Amount: 1},
		actEntry{Address: "space-mountain", AssetID: "asset-id-1", AssetLabel: "silver", Amount: 2},
	}

	wantOuts := []actEntry{
		actEntry{AccountID: "account-id-2", AccountLabel: "alice", AssetID: "asset-id-0", AssetLabel: "gold", Amount: 1},
		actEntry{AccountID: "account-id-2", AccountLabel: "alice", AssetID: "asset-id-1", AssetLabel: "silver", Amount: 2},
		actEntry{AccountID: "account-id-1", AccountLabel: "bob", AssetID: "asset-id-2", AssetLabel: "frankincense", Amount: 3},
		actEntry{AccountID: "account-id-1", AccountLabel: "bob", AssetID: "asset-id-3", AssetLabel: "myrrh", Amount: 4},
		actEntry{Address: "teacups", AssetID: "asset-id-4", AssetLabel: "avocado", Amount: 5},
		actEntry{Address: "teacups", AssetID: "asset-id-5", AssetLabel: "mango", Amount: 6},
	}

	if !reflect.DeepEqual(gotIns, wantIns) {
		t.Errorf("input entries:\ngot:  %v\nwant: %v", gotIns, wantIns)
	}

	if !reflect.DeepEqual(gotOuts, wantOuts) {
		t.Errorf("output entries:\ngot:  %v\nwant: %v", gotOuts, wantOuts)
	}
}

func TestWriteManagerNodeActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	accounts := []string{"account-id-0", "account-id-1"}
	accountSet := make(map[string]bool)
	for _, b := range accounts {
		accountSet[b] = true
	}

	err := writeManagerNodeActivity(
		ctx,
		"manager-node-id-0", "tx-hash", []byte(`{"transaction_id":"dummy"}`),
		accounts,
	)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	gotAct, err := getTestActivity(ctx, "tx-hash", false)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	wantAct := map[string]actItem{"manager-node-id-0": actItem{TxHash: "dummy"}}
	if !reflect.DeepEqual(gotAct, wantAct) {
		t.Errorf("activity rows:\ngot:  %v\nwant: %v", gotAct, wantAct)
	}

	gotAccounts, err := getTestActivityAccounts(ctx, "tx-hash")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	wantAccounts := []string{"account-id-0", "account-id-1"}
	if !reflect.DeepEqual(gotAccounts, wantAccounts) {
		t.Errorf("accounts:\ngot:  %v\nwant: %v", gotAccounts, wantAccounts)
	}
}

func TestWriteIssuanceActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	err := writeIssuanceActivity(
		ctx,
		&actAsset{id: "asset-id-0", inID: "in-id-0"},
		"tx-hash",
		[]byte(`{"transaction_id": "dummy"}`),
	)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	gotAct, err := getTestActivity(ctx, "tx-hash", true)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	wantAct := map[string]actItem{"in-id-0": actItem{TxHash: "dummy"}}
	if !reflect.DeepEqual(gotAct, wantAct) {
		t.Errorf("activity rows:\ngot:  %v\nwant: %v", gotAct, wantAct)
	}

	gotAssets, err := getTestActivityAssets(ctx, "tx-hash")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	wantAssets := []string{"asset-id-0"}
	if !reflect.DeepEqual(gotAssets, wantAssets) {
		t.Errorf("assets:\ngot:  %v\nwant: %v", gotAssets, wantAssets)
	}
}

func getTestActivity(ctx context.Context, txHash string, issuance bool) (map[string]actItem, error) {
	relationID := "manager_node_id"
	table := "activity"
	if issuance {
		relationID = "issuer_node_id"
		table = "issuance_activity"
	}

	q := `
		SELECT ` + relationID + `, data
		FROM ` + table + `
		WHERE txid = $1
	`
	rows, err := pg.FromContext(ctx).Query(q, txHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]actItem)
	for rows.Next() {
		var (
			id   string
			data []byte
		)
		err := rows.Scan(&id, &data)
		if err != nil {
			return nil, err
		}

		var item actItem
		err = json.Unmarshal(data, &item)
		if err != nil {
			return nil, err
		}

		res[id] = item
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

func getTestActivityAccounts(ctx context.Context, txHash string) ([]string, error) {
	q := `
		SELECT array_agg(account_id ORDER BY account_id)
		FROM activity_accounts aa
		JOIN activity a ON aa.activity_id = a.id
		WHERE a.txid = $1
	`
	var res []string
	err := pg.FromContext(ctx).QueryRow(q, txHash).Scan((*pg.Strings)(&res))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func getTestActivityAssets(ctx context.Context, txHash string) ([]string, error) {
	q := `
		SELECT array_agg(asset_id ORDER BY asset_id)
		FROM issuance_activity_assets iaa
		JOIN issuance_activity ia ON iaa.issuance_activity_id = ia.id
		WHERE ia.txid = $1
	`
	var res []string
	err := pg.FromContext(ctx).QueryRow(q, txHash).Scan((*pg.Strings)(&res))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func mustHashFromStr(s string) bc.Hash {
	h, err := bc.ParseHash(s)
	if err != nil {
		panic(err)
	}
	return h
}

func stringsToRawJSON(strs ...string) []*json.RawMessage {
	var res []*json.RawMessage
	for _, s := range strs {
		b := json.RawMessage([]byte(s))
		res = append(res, &b)
	}
	return res
}

func withStack(err error) string {
	s := err.Error()
	for _, frame := range errors.Stack(err) {
		s += "\n" + frame.String()
	}
	return s
}
