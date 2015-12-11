package appdb

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
)

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
		(id, issuer_node_id, key_index, redeem_script, issuance_script, label)
	VALUES
		('asset-id-0', 'in-id-0', 0, '\x'::bytea, '\x'::bytea, 'asset-0'),
		('asset-id-1', 'in-id-0', 1, '\x'::bytea, '\x'::bytea, 'asset-1'),
		('asset-id-2', 'in-id-1', 0, '\x'::bytea, '\x'::bytea, 'asset-2');

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

// Some test addresses and PK scripts.
var testAddrs = []struct {
	addr   string
	script string
}{
	{"33JaS6naDzZM45imNgNTX93374rL4cn4Na", "a91411b1d274c20532f6b5611d90fa6d854e88fe911687"},
	{"3EufaWajf5FYtpmpgVCgJEoceFWogyjGih", "a91490fe0f28833af4c9d9194eaa0b5b3aae787a177287"},
	{"3FgNFea8fmCSXWfrcCJps5j3FeZ8f2BNde", "a914997250aca70e0d3b9007489ae28dc6760a7a7d7487"},
	{"3M1zHhUvi8vfmTgdB2QdtqLBwAZWZR2h7F", "a914d400ed9954c6a1f19e9e224d751ab5bc38c56a1487"},
	{"3P8h3P1gCfbUQD13YEKcH3kMHfctcpUo4B", "a914eb35b1c812f943883b46ac48580a93012b5aa1aa87"},
}

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
	txTime := time.Now().UTC()

	issuanceTx := bc.NewTx(bc.TxData{
		// Issuances have a single prevout with an empty tx hash.
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Index: bc.InvalidOutputIndex}}},
		Outputs: []*bc.TxOutput{{}, {}, {}},
	})

	transferTx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{{
			Previous: bc.Outpoint{Hash: mustHashFromStr("4786c29077265138e00a8fce822c5fb998c0ce99df53d939bb53d81bca5aa426"), Index: 0},
		}},
		Outputs: []*bc.TxOutput{{}, {}, {}},
	})

	examples := []struct {
		tx          *bc.Tx
		fixture     string
		outIsChange map[int]bool

		wantManagerNodeActivity map[string]actItem
		wantAccounts            []string
		wantIssuanceActivity    map[string]actItem
		wantAssets              []string
	}{
		// Issuance
		{
			tx: issuanceTx,
			fixture: `
				INSERT INTO pool_txs (tx_hash, data)
				VALUES ('` + issuanceTx.Hash.String() + `', '\x'::bytea);

				INSERT INTO utxos (
					tx_hash, pool_tx_hash, index,
					asset_id, amount, script,
					addr_index, account_id, manager_node_id, confirmed
				) VALUES (
					'` + issuanceTx.Hash.String() + `', '` + issuanceTx.Hash.String() + `', 0,
					'asset-id-0', 1, decode('` + testAddrs[0].script + `', 'hex'),
					0, 'account-id-0', 'manager-node-id-0', FALSE
				), (
					'` + issuanceTx.Hash.String() + `', '` + issuanceTx.Hash.String() + `', 1,
					'asset-id-0', 2,  decode('` + testAddrs[1].script + `', 'hex'),
					0, 'account-id-2', 'manager-node-id-1', FALSE
				), (
					'` + issuanceTx.Hash.String() + `', '` + issuanceTx.Hash.String() + `', 2,
					'asset-id-0', 3,  decode('` + testAddrs[2].script + `', 'hex'),
					0, '', '', FALSE
				);
			`,
			outIsChange: make(map[int]bool),

			wantManagerNodeActivity: map[string]actItem{
				"manager-node-id-0": actItem{
					TxHash: issuanceTx.Hash.String(),
					Time:   txTime,
					Inputs: []actEntry{},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, AccountID: "account-id-0", AccountLabel: "account-0"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, Address: testAddrs[1].addr},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 3, Address: testAddrs[2].addr},
					},
				},
				"manager-node-id-1": actItem{
					TxHash: issuanceTx.Hash.String(),
					Time:   txTime,
					Inputs: []actEntry{},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, AccountID: "account-id-2", AccountLabel: "account-2"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, Address: testAddrs[0].addr},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 3, Address: testAddrs[2].addr},
					},
				},
			},
			wantAccounts: []string{"account-id-0", "account-id-2"},
			wantIssuanceActivity: map[string]actItem{
				"in-id-0": actItem{
					TxHash: issuanceTx.Hash.String(),
					Time:   txTime,
					Inputs: []actEntry{},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, AccountID: "account-id-0", AccountLabel: "account-0"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, AccountID: "account-id-2", AccountLabel: "account-2"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 3, Address: testAddrs[2].addr},
					},
				},
			},
			wantAssets: []string{"asset-id-0"},
		},

		// Transfer with change
		{
			tx: transferTx,
			fixture: `
				INSERT INTO utxos (
					tx_hash, index,
					asset_id, amount, script,
					addr_index, account_id, manager_node_id, confirmed,
					block_hash, block_height
				) VALUES (
					'4786c29077265138e00a8fce822c5fb998c0ce99df53d939bb53d81bca5aa426', 0,
					'asset-id-0', 6, decode('` + testAddrs[0].script + `', 'hex'),
					0, 'account-id-0', 'manager-node-id-0', TRUE,
					'bh1', 1
				);

				INSERT INTO pool_txs (tx_hash, data)
				VALUES ('` + transferTx.Hash.String() + `', '\x'::bytea);

				INSERT INTO utxos (
					tx_hash, pool_tx_hash, index,
					asset_id, amount, script,
					addr_index, account_id, manager_node_id, confirmed
				) VALUES (
					'` + transferTx.Hash.String() + `', '` + transferTx.Hash.String() + `', 0,
					'asset-id-0', 1, decode('` + testAddrs[1].script + `', 'hex'),
					0, 'account-id-2', 'manager-node-id-1', FALSE
				), (
					'` + transferTx.Hash.String() + `', '` + transferTx.Hash.String() + `', 1,
					'asset-id-0', 2, decode('` + testAddrs[2].script + `', 'hex'),
					0, 'account-id-0', 'manager-node-id-0', FALSE
				), (
					'` + transferTx.Hash.String() + `', '` + transferTx.Hash.String() + `', 2,
					'asset-id-0', 3, decode('` + testAddrs[3].script + `', 'hex'),
					0, '', '', FALSE
				);
			`,
			outIsChange: map[int]bool{1: true},

			wantManagerNodeActivity: map[string]actItem{
				"manager-node-id-0": actItem{
					TxHash: transferTx.Hash.String(),
					Time:   txTime,
					Inputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 4, AccountID: "account-id-0", AccountLabel: "account-0"},
					},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, Address: testAddrs[1].addr},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 3, Address: testAddrs[3].addr},
					},
				},
				"manager-node-id-1": actItem{
					TxHash: transferTx.Hash.String(),
					Time:   txTime,
					Inputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 6, Address: testAddrs[0].addr},
					},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, AccountID: "account-id-2", AccountLabel: "account-2"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, Address: testAddrs[2].addr},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 3, Address: testAddrs[3].addr},
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
			txHash := ex.tx.Hash.String()

			dbtx := pgtest.TxWithSQL(t, writeActivityFix, ex.fixture)
			ctx := pg.NewContext(context.Background(), dbtx)
			defer dbtx.Rollback()

			err := WriteActivity(ctx, ex.tx, ex.outIsChange, txTime)
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
	tx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{
			{
				Previous: bc.Outpoint{
					Hash:  mustHashFromStr("0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098"),
					Index: 0,
				},
			},
			{
				Previous: bc.Outpoint{
					Hash:  mustHashFromStr("3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df"),
					Index: 1,
				},
			},
			{
				Previous: bc.Outpoint{
					Hash:  mustHashFromStr("7de759a6e917f941e8da7c30e6ad8a3d85a4f508d5bbed4fe80244271754eaef"),
					Index: 0,
				},
			},
		},
		Outputs: []*bc.TxOutput{{}, {}},
	})

	dbtx := pgtest.TxWithSQL(t, writeActivityFix, `
		INSERT INTO utxos (
			tx_hash, index,
			asset_id, amount, addr_index, script,
			account_id, manager_node_id, confirmed,
			block_hash, block_height
		) VALUES (
			'0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 0,
			'asset-id-0', 1, 0, decode('`+testAddrs[0].script+`', 'hex'),
			'account-id-0', 'manager-node-id-0', TRUE,
			'bh1', 1
		), (
			'3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 1,
			'asset-id-0', 2, 1, decode('`+testAddrs[1].script+`', 'hex'),
			'account-id-1', 'manager-node-id-0', TRUE,
			'bh1', 1
		);

		INSERT INTO pool_txs
			(tx_hash, data)
		VALUES
			('7de759a6e917f941e8da7c30e6ad8a3d85a4f508d5bbed4fe80244271754eaef', '\x'::bytea),
			('`+tx.Hash.String()+`', '\x'::bytea);

		INSERT INTO utxos (
			tx_hash, pool_tx_hash, index,
			asset_id, amount, addr_index, script,
			account_id, manager_node_id, confirmed
		) VALUES (
			'7de759a6e917f941e8da7c30e6ad8a3d85a4f508d5bbed4fe80244271754eaef', '7de759a6e917f941e8da7c30e6ad8a3d85a4f508d5bbed4fe80244271754eaef', 0,
			'asset-id-1', 3, 0, decode('`+testAddrs[2].script+`', 'hex'),
			'account-id-2', 'manager-node-id-2', FALSE
		), (
			'`+tx.Hash.String()+`', '`+tx.Hash.String()+`', 0,
			'asset-id-0', 3, 1, decode('`+testAddrs[3].script+`', 'hex'),
			'account-id-3', 'manager-node-id-3', FALSE
		), (
			'`+tx.Hash.String()+`', '`+tx.Hash.String()+`', 1,
			'asset-id-1', 3, 0, decode('`+testAddrs[4].script+`', 'hex'),
			'account-id-4', 'manager-node-id-4', FALSE
		);
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	gotIns, gotOuts, err := GetActUTXOs(ctx, tx)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	wantIns := []*ActUTXO{
		{
			AssetID:       "asset-id-0",
			Amount:        1,
			ManagerNodeID: "manager-node-id-0",
			AccountID:     "account-id-0",
			Addr:          testAddrs[0].addr,
			Script:        mustDecodeHex(testAddrs[0].script),
		},
		{
			AssetID:       "asset-id-0",
			Amount:        2,
			ManagerNodeID: "manager-node-id-0",
			AccountID:     "account-id-1",
			Addr:          testAddrs[1].addr,
			Script:        mustDecodeHex(testAddrs[1].script),
		},
		{
			AssetID:       "asset-id-1",
			Amount:        3,
			ManagerNodeID: "manager-node-id-2",
			AccountID:     "account-id-2",
			Addr:          testAddrs[2].addr,
			Script:        mustDecodeHex(testAddrs[2].script),
		},
	}

	wantOuts := []*ActUTXO{
		{
			AssetID:       "asset-id-0",
			Amount:        3,
			ManagerNodeID: "manager-node-id-3",
			AccountID:     "account-id-3",
			Addr:          testAddrs[3].addr,
			Script:        mustDecodeHex(testAddrs[3].script),
		},
		{
			AssetID:       "asset-id-1",
			Amount:        3,
			ManagerNodeID: "manager-node-id-4",
			AccountID:     "account-id-4",
			Addr:          testAddrs[4].addr,
			Script:        mustDecodeHex(testAddrs[4].script),
		},
	}

	if !reflect.DeepEqual(gotIns, wantIns) {
		t.Errorf("inputs:\ngot:  %v\nwant: %v", gotIns, wantIns)
	}

	if !reflect.DeepEqual(gotOuts, wantOuts) {
		t.Errorf("outputs:\ngot:  %v\nwant: %v", gotOuts, wantOuts)
	}
}

func TestGetActUTXOsIssuance(t *testing.T) {
	tx := bc.NewTx(bc.TxData{
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Index: bc.InvalidOutputIndex}}},
		Outputs: []*bc.TxOutput{{}, {}},
	})

	dbtx := pgtest.TxWithSQL(t, writeActivityFix, `
		INSERT INTO pool_txs
			(tx_hash, data)
		VALUES
			('`+tx.Hash.String()+`', '\x'::bytea);

		INSERT INTO utxos (
			tx_hash, pool_tx_hash, index,
			asset_id, amount, addr_index, script,
			account_id, manager_node_id, confirmed
		) VALUES (
			'`+tx.Hash.String()+`', '`+tx.Hash.String()+`', 0,
			'asset-id-0', 1, 0, decode('`+testAddrs[0].script+`', 'hex'),
			'account-id-0', 'manager-node-id-0', FALSE
		), (
			'`+tx.Hash.String()+`', '`+tx.Hash.String()+`', 1,
			'asset-id-0', 2, 1, decode('`+testAddrs[1].script+`', 'hex'),
			'account-id-1', 'manager-node-id-1', FALSE
		);
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	gotIns, gotOuts, err := GetActUTXOs(ctx, tx)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	var wantIns []*ActUTXO
	wantOuts := []*ActUTXO{
		{
			AssetID:       "asset-id-0",
			Amount:        1,
			ManagerNodeID: "manager-node-id-0",
			AccountID:     "account-id-0",
			Addr:          testAddrs[0].addr,
			Script:        mustDecodeHex(testAddrs[0].script),
		},
		{
			AssetID:       "asset-id-0",
			Amount:        2,
			ManagerNodeID: "manager-node-id-1",
			AccountID:     "account-id-1",
			Addr:          testAddrs[1].addr,
			Script:        mustDecodeHex(testAddrs[1].script),
		},
	}

	if !reflect.DeepEqual(gotIns, wantIns) {
		t.Errorf("inputs:\ngot:  %v\nwant: %v", gotIns, wantIns)
	}

	if !reflect.DeepEqual(gotOuts, wantOuts) {
		t.Errorf("outputs:\ngot:  %v\nwant: %v", gotOuts, wantOuts)
	}
}

func TestMarkChangeOuts(t *testing.T) {
	const fix = `
		INSERT INTO projects
			(id, name)
		VALUES
			('proj-id-0', 'proj-0');

		INSERT INTO manager_nodes
			(id, project_id, key_index, label, current_rotation, sigs_required)
		VALUES
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0', 'rot-id-0', 1);

		INSERT INTO accounts
			(id, manager_node_id, key_index, label)
		VALUES
			('account-id-0', 'manager-node-id-0', 0, 'account-0');

		INSERT INTO addresses
			(address, is_change, manager_node_id, account_id, keyset, key_index, redeem_script, pk_script)
		VALUES
			('addr-0', true, 'manager-node-id-0', 'account-id-0', '{}', 0, '\x'::bytea, '\x'::bytea),
			('addr-1', false, 'manager-node-id-0', 'account-id-0', '{}', 1, '\x'::bytea, '\x'::bytea);
	`

	withContext(t, fix, func(ctx context.Context) {
		utxos := []*ActUTXO{
			{Addr: "addr-0"},
			{Addr: "addr-1"},
			{Addr: "addr-2"},
			{Addr: "addr-3"},
		}
		outIsChange := map[int]bool{2: true}

		err := markChangeOuts(ctx, utxos, outIsChange)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []*ActUTXO{
			{Addr: "addr-0", IsChange: true},
			{Addr: "addr-1"},
			{Addr: "addr-2", IsChange: true},
			{Addr: "addr-3"},
		}

		if !reflect.DeepEqual(utxos, want) {
			t.Errorf("change flags:\ngot:  %v\nwant: %v", utxos, want)
		}
	})
}

func TestGetIDsFromUTXOs(t *testing.T) {
	utxos := []*ActUTXO{
		{AssetID: "asset-id-3"},
		{AccountID: "account-id-2", AssetID: "asset-id-2", ManagerNodeID: "manager-node-id-1"},
		{AccountID: "account-id-1", AssetID: "asset-id-2", ManagerNodeID: "manager-node-id-1"},
		{AccountID: "account-id-0", AssetID: "asset-id-2", ManagerNodeID: "manager-node-id-0"},
		{AccountID: "account-id-0", AssetID: "asset-id-1", ManagerNodeID: "manager-node-id-0"},
		{AccountID: "account-id-0", AssetID: "asset-id-0", ManagerNodeID: "manager-node-id-0"},
		{AccountID: "account-id-0", AssetID: "asset-id-0", ManagerNodeID: "manager-node-id-0"},
	}

	gAssetIDs, gAccountIDs, gManagerNodeIDs, gManagerNodeAccounts := getIDsFromUTXOs(utxos)

	wAssetIDs := []string{"asset-id-0", "asset-id-1", "asset-id-2", "asset-id-3"}
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
		want     []*ActAsset
	}{
		{
			[]string{"asset-id-0", "asset-id-2"},
			[]*ActAsset{
				{ID: "asset-id-0", Label: "asset-0", IssuerNodeID: "in-id-0", ProjID: "proj-id-0"},
				{ID: "asset-id-2", Label: "asset-2", IssuerNodeID: "in-id-1", ProjID: "proj-id-0"},
			},
		},
		{
			[]string{"asset-id-1"},
			[]*ActAsset{
				{ID: "asset-id-1", Label: "asset-1", IssuerNodeID: "in-id-0", ProjID: "proj-id-0"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.assetIDs)

		got, err := GetActAssets(ctx, ex.assetIDs)
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
		want       []*ActAccount
	}{
		{
			[]string{"account-id-0", "account-id-2"},
			[]*ActAccount{
				{ID: "account-id-0", Label: "account-0", ManagerNodeID: "manager-node-id-0", ProjID: "proj-id-0"},
				{ID: "account-id-2", Label: "account-2", ManagerNodeID: "manager-node-id-1", ProjID: "proj-id-0"},
			},
		},
		{
			[]string{"account-id-1"},
			[]*ActAccount{
				{ID: "account-id-1", Label: "account-1", ManagerNodeID: "manager-node-id-0", ProjID: "proj-id-0"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.accountIDs)

		got, err := GetActAccounts(ctx, ex.accountIDs)
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
		ins, outs       []*ActUTXO
		visibleAccounts []string
		want            txRawActivity
	}{
		// Simple transfer from Alice's perspective
		{
			ins: []*ActUTXO{
				{AccountID: "alice-account-id-0", AssetID: "gold", Amount: 10, Addr: "alice-addr-id-0"},
			},
			outs: []*ActUTXO{
				{AccountID: "bob-account-id-0", AssetID: "gold", Amount: 10, Addr: "bob-addr-id-0"},
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
			ins: []*ActUTXO{
				{AssetID: "gold", Amount: 10, AccountID: "alice-account-id-0", Addr: "alice-addr-id-0"},
			},
			outs: []*ActUTXO{
				{AssetID: "gold", Amount: 10, AccountID: "bob-account-id-0", Addr: "bob-addr-id-0"},
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
			ins: []*ActUTXO{
				{AssetID: "gold", Amount: 20, AccountID: "alice-account-id-0", Addr: "alice-addr-id-0"},
				{AssetID: "silver", Amount: 10, AccountID: "bob-account-id-0", Addr: "bob-addr-id-0"},
				{AssetID: "silver", Amount: 10, AccountID: "bob-account-id-0", Addr: "bob-addr-id-1"},
			},
			outs: []*ActUTXO{
				{AssetID: "silver", Amount: 15, AccountID: "alice-account-id-0", Addr: "alice-addr-id-1"},
				{AssetID: "gold", Amount: 5, AccountID: "bob-account-id-0", Addr: "bob-addr-id-2"},

				{AssetID: "gold", Amount: 15, AccountID: "alice-account-id-0", Addr: "alice-addr-id-2", IsChange: true},
				{AssetID: "silver", Amount: 5, AccountID: "bob-account-id-0", Addr: "bob-addr-id-3", IsChange: true},
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
			ins: []*ActUTXO{
				{AssetID: "gold", Amount: 20, AccountID: "alice-account-id-0", Addr: "alice-addr-id-0"},
				{AssetID: "silver", Amount: 10, AccountID: "bob-account-id-0", Addr: "bob-addr-id-0"},
				{AssetID: "silver", Amount: 10, AccountID: "bob-account-id-0", Addr: "bob-addr-id-1"},
			},
			outs: []*ActUTXO{
				{AssetID: "silver", Amount: 15, AccountID: "alice-account-id-0", Addr: "alice-addr-id-1"},
				{AssetID: "gold", Amount: 5, AccountID: "bob-account-id-0", Addr: "bob-addr-id-2"},

				{AssetID: "gold", Amount: 15, AccountID: "alice-account-id-0", Addr: "alice-addr-id-2", IsChange: true},
				{AssetID: "silver", Amount: 5, AccountID: "bob-account-id-0", Addr: "bob-addr-id-3", IsChange: true},
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
		&ActAsset{ID: "asset-id-0", IssuerNodeID: "in-id-0"},
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

func mustDecodeHex(str string) []byte {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return bytes
}
