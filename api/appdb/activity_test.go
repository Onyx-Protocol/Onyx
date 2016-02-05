package appdb_test

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"

	. "chain/api/appdb"
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
var testAddrs = []string{
	"a91411b1d274c20532f6b5611d90fa6d854e88fe911687",
	"a91490fe0f28833af4c9d9194eaa0b5b3aae787a177287",
	"a914997250aca70e0d3b9007489ae28dc6760a7a7d7487",
	"a914d400ed9954c6a1f19e9e224d751ab5bc38c56a1487",
	"a914eb35b1c812f943883b46ac48580a93012b5aa1aa87",
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

	ctx := pgtest.NewContext(t, writeActivityFix, `
		INSERT INTO utxos (
			tx_hash, index,
			asset_id, amount, addr_index, script,
			account_id, manager_node_id, confirmed,
			block_hash, block_height
		) VALUES (
			'0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 0,
			'asset-id-0', 1, 0, decode('`+testAddrs[0]+`', 'hex'),
			'account-id-0', 'manager-node-id-0', TRUE,
			'bh1', 1
		), (
			'3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 1,
			'asset-id-0', 2, 1, decode('`+testAddrs[1]+`', 'hex'),
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
			'asset-id-1', 3, 0, decode('`+testAddrs[2]+`', 'hex'),
			'account-id-2', 'manager-node-id-2', FALSE
		), (
			'`+tx.Hash.String()+`', '`+tx.Hash.String()+`', 0,
			'asset-id-0', 3, 1, decode('`+testAddrs[3]+`', 'hex'),
			'account-id-3', 'manager-node-id-3', FALSE
		), (
			'`+tx.Hash.String()+`', '`+tx.Hash.String()+`', 1,
			'asset-id-1', 3, 0, decode('`+testAddrs[4]+`', 'hex'),
			'account-id-4', 'manager-node-id-4', FALSE
		);
	`)
	defer pgtest.Finish(ctx)

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
			Script:        mustDecodeHex(testAddrs[0]),
		},
		{
			AssetID:       "asset-id-0",
			Amount:        2,
			ManagerNodeID: "manager-node-id-0",
			AccountID:     "account-id-1",
			Script:        mustDecodeHex(testAddrs[1]),
		},
		{
			AssetID:       "asset-id-1",
			Amount:        3,
			ManagerNodeID: "manager-node-id-2",
			AccountID:     "account-id-2",
			Script:        mustDecodeHex(testAddrs[2]),
		},
	}

	wantOuts := []*ActUTXO{
		{
			AssetID:       "asset-id-0",
			Amount:        3,
			ManagerNodeID: "manager-node-id-3",
			AccountID:     "account-id-3",
			Script:        mustDecodeHex(testAddrs[3]),
		},
		{
			AssetID:       "asset-id-1",
			Amount:        3,
			ManagerNodeID: "manager-node-id-4",
			AccountID:     "account-id-4",
			Script:        mustDecodeHex(testAddrs[4]),
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

	ctx := pgtest.NewContext(t, writeActivityFix, `
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
			'asset-id-0', 1, 0, decode('`+testAddrs[0]+`', 'hex'),
			'account-id-0', 'manager-node-id-0', FALSE
		), (
			'`+tx.Hash.String()+`', '`+tx.Hash.String()+`', 1,
			'asset-id-0', 2, 1, decode('`+testAddrs[1]+`', 'hex'),
			'account-id-1', 'manager-node-id-1', FALSE
		);
	`)
	defer pgtest.Finish(ctx)

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
			Script:        mustDecodeHex(testAddrs[0]),
		},
		{
			AssetID:       "asset-id-0",
			Amount:        2,
			ManagerNodeID: "manager-node-id-1",
			AccountID:     "account-id-1",
			Script:        mustDecodeHex(testAddrs[1]),
		},
	}

	if !reflect.DeepEqual(gotIns, wantIns) {
		t.Errorf("inputs:\ngot:  %v\nwant: %v", gotIns, wantIns)
	}

	if !reflect.DeepEqual(gotOuts, wantOuts) {
		t.Errorf("outputs:\ngot:  %v\nwant: %v", gotOuts, wantOuts)
	}
}

func TestGetActAssets(t *testing.T) {
	ctx := pgtest.NewContext(t, writeActivityFix)
	defer pgtest.Finish(ctx)

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
	ctx := pgtest.NewContext(t, writeActivityFix)
	defer pgtest.Finish(ctx)

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
