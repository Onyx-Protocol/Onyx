package appdb

import (
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/errors"
	"chain/fedchain-sandbox/wire"
)

const writeActivityFix = `
	INSERT INTO applications
		(id, name)
	VALUES
		('app-id-0', 'app-0');

	INSERT INTO wallets
		(id, application_id, key_index, label)
	VALUES
		('wallet-id-0', 'app-id-0', 0, 'wallet-0'),
		('wallet-id-1', 'app-id-0', 0, 'wallet-1');

	INSERT INTO buckets
		(id, wallet_id, key_index, label)
	VALUES
		('bucket-id-0', 'wallet-id-0', 0, 'bucket-0'),
		('bucket-id-1', 'wallet-id-0', 1, 'bucket-1'),
		('bucket-id-2', 'wallet-id-1', 0, 'bucket-2');

	INSERT INTO addresses
		(id, wallet_id, bucket_id, keyset, key_index, address, redeem_script, pk_script, is_change)
	VALUES
		('addr-id-0', 'wallet-id-0', 'bucket-id-0', '{}', 0, 'addr-0', '{}', '{}', false),
		('addr-id-1', 'wallet-id-0', 'bucket-id-0', '{}', 1, 'addr-1', '{}', '{}', true),
		('addr-id-2', 'wallet-id-0', 'bucket-id-1', '{}', 0, 'addr-2', '{}', '{}', false),
		('addr-id-3', 'wallet-id-0', 'bucket-id-1', '{}', 1, 'addr-3', '{}', '{}', true),
		('addr-id-4', 'wallet-id-1', 'bucket-id-2', '{}', 0, 'addr-4', '{}', '{}', false),
		('addr-id-5', 'wallet-id-1', 'bucket-id-2', '{}', 1, 'addr-5', '{}', '{}', true);

	INSERT INTO asset_groups
		(id, application_id, key_index, label, keyset)
	VALUES
		('ag-id-0', 'app-id-0', 0, 'ag-0', '{}'),
		('ag-id-1', 'app-id-0', 1, 'ag-1', '{}');

	INSERT INTO assets
		(id, asset_group_id, key_index, redeem_script, label)
	VALUES
		('asset-id-0', 'ag-id-0', 0, '{}', 'asset-0'),
		('asset-id-1', 'ag-id-0', 1, '{}', 'asset-1'),
		('asset-id-2', 'ag-id-1', 0, '{}', 'asset-2');
`

const sampleActivityFixture = `
	INSERT INTO wallets (id, application_id, label, current_rotation, key_index)
		VALUES('w0', 'app-id-0', '', 'c0', 0);
	INSERT INTO activity (id, wallet_id, data, txid)
		VALUES('act0', 'w0', '{"outputs":"boop"}', 'tx0');
`

func TestWalletActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, sampleActivityFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	activity, last, err := WalletActivity(ctx, "w0", "act2", 1) // act2 would be a newer item than act1
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

func TestWalletActivityLimit(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, sampleActivityFixture, `
		INSERT INTO activity (id, wallet_id, data, txid)
			VALUES
				('act1', 'w0', '{"outputs":"coop"}', 'tx1'),
				('act2', 'w0', '{"outputs":"doop"}', 'tx2'),
				('act3', 'w0', '{"outputs":"foop"}', 'tx3');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	activity, last, err := WalletActivity(ctx, "w0", "act4", 2) // act4 would be a newer item than act1
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

func TestBucketActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, sampleActivityFixture, `
		INSERT INTO buckets (id, wallet_id, key_index) VALUES('b0', 'w0', 0);
		INSERT INTO activity_buckets VALUES ('act0', 'b0');
	`)

	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	activity, last, err := BucketActivity(ctx, "b0", "act1", 1)
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

func TestBucketActivityLimit(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, sampleActivityFixture, `
		INSERT INTO activity (id, wallet_id, data, txid)
			VALUES
			('act1', 'w0', '{"outputs":"coop"}', 'tx1'),
			('act2', 'w0', '{"outputs":"doop"}', 'tx2'),
			('act3', 'w0', '{"outputs":"foop"}', 'tx3');
		INSERT INTO buckets (id, wallet_id, key_index) VALUES('b0', 'w0', 0);
		INSERT INTO activity_buckets VALUES
			('act0', 'b0'),
			('act1', 'b0'),
			('act2', 'b0'),
			('act3', 'b0');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	activity, last, err := BucketActivity(ctx, "b0", "act4", 2) // act4 would be a newer item than act1
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

func TestAssetGroupActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix, `
		INSERT INTO issuance_activity
			(id, asset_group_id, data, txid)
		VALUES
			('ia-id-0', 'ag-id-0', '{"transaction_id": "tx-id-0"}', 'tx-id-0'),
			('ia-id-1', 'ag-id-1', '{"transaction_id": "tx-id-1"}', 'tx-id-1'),
			('ia-id-2', 'ag-id-0', '{"transaction_id": "tx-id-2"}', 'tx-id-2');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		agID     string
		wantAct  []*json.RawMessage
		wantLast string
	}{
		{
			"ag-id-0",
			stringsToRawJSON(
				`{"transaction_id": "tx-id-2"}`,
				`{"transaction_id": "tx-id-0"}`,
			),
			"ia-id-0",
		},
		{
			"ag-id-1",
			stringsToRawJSON(
				`{"transaction_id": "tx-id-1"}`,
			),
			"ia-id-1",
		},
	}

	for _, ex := range examples {
		t.Log("Example", ex.agID)

		gotAct, gotLast, err := AssetGroupActivity(ctx, ex.agID, "", 50)
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
			(id, asset_group_id, data, txid)
		VALUES
			('ia-id-0', 'ag-id-0', '{"transaction_id": "tx-id-0"}', 'tx-id-0'),
			('ia-id-1', 'ag-id-1', '{"transaction_id": "tx-id-1"}', 'tx-id-1'),
			('ia-id-2', 'ag-id-0', '{"transaction_id": "tx-id-2"}', 'tx-id-2');

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
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, sampleActivityFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	activity, err := WalletTxActivity(ctx, "w0", "tx0")
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if string(*activity) != `{"outputs":"boop"}` {
		t.Fatalf("want={outputs: boop}, got=%s", *activity)
	}

	_, err = WalletTxActivity(ctx, "w0", "txDoesNotExist")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Fatalf("want=%v got=%v", pg.ErrUserInputNotFound, err)
	}
}

// TestWriteActivity is an integration test for WriteActivity. Only basic use
// cases are covered here. Edge cases are covered in unit tests that follow.
func TestWriteActivity(t *testing.T) {
	// Mock transactions for creating prevouts.
	prevTxA := wire.NewMsgTx()
	prevTxA.AddTxOut(wire.NewTxOut(wire.Hash20{0}, 123, nil))
	prevTxB := wire.NewMsgTx()
	prevTxB.AddTxOut(wire.NewTxOut(wire.Hash20{1}, 456, nil))

	txTime := time.Now().UTC()

	examples := []struct {
		tx                   *wire.MsgTx
		fixture              string
		wantWalletActivity   map[string]actItem
		wantBuckets          []string
		wantIssuanceActivity map[string]actItem
		wantAssets           []string
	}{
		// Issuance
		{
			tx: &wire.MsgTx{
				// Issuances have a single prevout with an empty tx hash.
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: wire.OutPoint{Hash: wire.Hash32{}},
				}},
				// The content of the outs is irrelevant for the test. Issuances
				// require at least one output.
				TxOut: []*wire.TxOut{{AssetID: wire.Hash20{}}},
			},
			fixture: `
				INSERT INTO utxos
					(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
				VALUES
					('0282a32a77d3358b28f06134cba121e5c54b205fe9935bfbb06076169a4e89db', 0, 'asset-id-0', 1, 'addr-id-0', 'bucket-id-0', 'wallet-id-0'),
					('0282a32a77d3358b28f06134cba121e5c54b205fe9935bfbb06076169a4e89db', 1, 'asset-id-0', 2, 'addr-id-4', 'bucket-id-2', 'wallet-id-1');
			`,
			wantWalletActivity: map[string]actItem{
				"wallet-id-0": actItem{
					TxHash: "0282a32a77d3358b28f06134cba121e5c54b205fe9935bfbb06076169a4e89db",
					Time:   txTime,
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, BucketID: "bucket-id-0", BucketLabel: "bucket-0"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, Address: "addr-4"},
					},
				},
				"wallet-id-1": actItem{
					TxHash: "0282a32a77d3358b28f06134cba121e5c54b205fe9935bfbb06076169a4e89db",
					Time:   txTime,
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, BucketID: "bucket-id-2", BucketLabel: "bucket-2"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, Address: "addr-0"},
					},
				},
			},
			wantBuckets: []string{"bucket-id-0", "bucket-id-2"},
			wantIssuanceActivity: map[string]actItem{
				"ag-id-0": actItem{
					TxHash: "0282a32a77d3358b28f06134cba121e5c54b205fe9935bfbb06076169a4e89db",
					Time:   txTime,
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, BucketID: "bucket-id-0", BucketLabel: "bucket-0"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, BucketID: "bucket-id-2", BucketLabel: "bucket-2"},
					},
				},
			},
			wantAssets: []string{"asset-id-0"},
		},

		// Transfer with change
		{
			tx: &wire.MsgTx{
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: wire.OutPoint{Hash: mustHash32FromStr("4786c29077265138e00a8fce822c5fb998c0ce99df53d939bb53d81bca5aa426"), Index: 0},
				}},
				// Outputs can be nil, since they are retrieved from the database via txid
			},
			fixture: `
				INSERT INTO utxos
					(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
				VALUES
					('4786c29077265138e00a8fce822c5fb998c0ce99df53d939bb53d81bca5aa426', 0, 'asset-id-0', 3, 'addr-id-0', 'bucket-id-0', 'wallet-id-0'),
					('db49adbf4b456581d39b610b2e422e21807086c108d01c33363c2c488dc02b12', 0, 'asset-id-0', 1, 'addr-id-4', 'bucket-id-2', 'wallet-id-1'),
					('db49adbf4b456581d39b610b2e422e21807086c108d01c33363c2c488dc02b12', 1, 'asset-id-0', 2, 'addr-id-1', 'bucket-id-0', 'wallet-id-0');
			`,
			wantWalletActivity: map[string]actItem{
				"wallet-id-0": actItem{
					TxHash: "db49adbf4b456581d39b610b2e422e21807086c108d01c33363c2c488dc02b12",
					Time:   txTime,
					Inputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, BucketID: "bucket-id-0", BucketLabel: "bucket-0"},
					},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, Address: "addr-4"},
					},
				},
				"wallet-id-1": actItem{
					TxHash: "db49adbf4b456581d39b610b2e422e21807086c108d01c33363c2c488dc02b12",
					Time:   txTime,
					Inputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 3, Address: "addr-0"},
					},
					Outputs: []actEntry{
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 1, BucketID: "bucket-id-2", BucketLabel: "bucket-2"},
						{AssetID: "asset-id-0", AssetLabel: "asset-0", Amount: 2, Address: "addr-1"},
					},
				},
			},
			wantBuckets:          []string{"bucket-id-0", "bucket-id-2"},
			wantIssuanceActivity: make(map[string]actItem),
		},
	}

	for i, ex := range examples {
		t.Log("Example", i)

		func() {
			txHash := ex.tx.TxSha().String()

			dbtx := pgtest.TxWithSQL(t, writeActivityFix, ex.fixture)
			ctx := pg.NewContext(context.Background(), dbtx)
			defer dbtx.Rollback()

			err := WriteActivity(ctx, ex.tx, txTime)
			if err != nil {
				t.Fatal("unexpected error", err)
			}

			gotWalletActivity, err := getTestActivity(ctx, txHash, false)
			if err != nil {
				t.Fatal("unexpected error", err)
			}

			if !reflect.DeepEqual(gotWalletActivity, ex.wantWalletActivity) {
				t.Errorf("wallet activity:\ngot:  %v\nwant: %v", gotWalletActivity, ex.wantWalletActivity)
			}

			gotBuckets, err := getTestActivityBuckets(ctx, txHash)
			if err != nil {
				t.Fatal("unexpected error", err)
			}

			if !reflect.DeepEqual(gotBuckets, ex.wantBuckets) {
				t.Errorf("buckets:\ngot:  %v\nwant: %v", gotBuckets, ex.wantBuckets)
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
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO utxos
			(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES
			('0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 0, 'a0', 100, 'addr0', 'b0', 'w0'),
			('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 1, 'a0', 50, 'addr1', 'b1', 'w0'),
			('0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 1, 'a1', 25, 'addr2', 'b2', 'w1');
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

	want := []*actUTXO{
		{assetID: "a0", amount: 100, addrID: "addr0", bucketID: "b0", walletID: "w0"},
		{assetID: "a0", amount: 50, addrID: "addr1", bucketID: "b1", walletID: "w0"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("utxos:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestGetActTxUTXOsByTx(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO utxos
			(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES
			('0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 0, 'a0', 100, 'addr0', 'b0', 'w0'),
			('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 1, 'a0', 50, 'addr1', 'b1', 'w0'),
			('0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 1, 'a1', 25, 'addr2', 'b2', 'w1');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		txHash string
		want   []*actUTXO
	}{
		{
			"0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098",
			[]*actUTXO{
				{assetID: "a0", amount: 100, addrID: "addr0", bucketID: "b0", walletID: "w0"},
				{assetID: "a1", amount: 25, addrID: "addr2", bucketID: "b2", walletID: "w1"},
			},
		},
		{
			"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df",
			[]*actUTXO{
				{assetID: "a0", amount: 50, addrID: "addr1", bucketID: "b1", walletID: "w0"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.txHash)

		got, err := getActUTXOsByTx(ctx, ex.txHash)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("utxos:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestGetIDsFromUTXOs(t *testing.T) {
	utxos := []*actUTXO{
		{assetID: "asset-id-2", addrID: "addr-id-3", bucketID: "bucket-id-2", walletID: "wallet-id-1"},
		{assetID: "asset-id-2", addrID: "addr-id-2", bucketID: "bucket-id-1", walletID: "wallet-id-1"},
		{assetID: "asset-id-2", addrID: "addr-id-1", bucketID: "bucket-id-0", walletID: "wallet-id-0"},
		{assetID: "asset-id-1", addrID: "addr-id-0", bucketID: "bucket-id-0", walletID: "wallet-id-0"},
		{assetID: "asset-id-0", addrID: "addr-id-0", bucketID: "bucket-id-0", walletID: "wallet-id-0"},
		{assetID: "asset-id-0", addrID: "addr-id-0", bucketID: "bucket-id-0", walletID: "wallet-id-0"},
	}

	gAssetIDs, gAddrIDs, gBucketIDs, gWalletIDs, gWalletBuckets := getIDsFromUTXOs(utxos)

	wAssetIDs := []string{"asset-id-0", "asset-id-1", "asset-id-2"}
	wAddrIDs := []string{"addr-id-0", "addr-id-1", "addr-id-2", "addr-id-3"}
	wBucketIDs := []string{"bucket-id-0", "bucket-id-1", "bucket-id-2"}
	wWalletIDs := []string{"wallet-id-0", "wallet-id-1"}
	wWalletBuckets := map[string][]string{
		"wallet-id-0": []string{"bucket-id-0"},
		"wallet-id-1": []string{"bucket-id-1", "bucket-id-2"},
	}

	if !reflect.DeepEqual(gAssetIDs, wAssetIDs) {
		t.Errorf("assetIDs:\ngot:  %v\nwant: %v", gAssetIDs, wAssetIDs)
	}
	if !reflect.DeepEqual(gAddrIDs, wAddrIDs) {
		t.Errorf("assetIDs:\ngot:  %v\nwant: %v", gAddrIDs, wAddrIDs)
	}
	if !reflect.DeepEqual(gBucketIDs, wBucketIDs) {
		t.Errorf("assetIDs:\ngot:  %v\nwant: %v", gBucketIDs, wBucketIDs)
	}
	if !reflect.DeepEqual(gWalletIDs, wWalletIDs) {
		t.Errorf("assetIDs:\ngot:  %v\nwant: %v", gWalletIDs, wWalletIDs)
	}
	if !reflect.DeepEqual(gWalletBuckets, wWalletBuckets) {
		t.Errorf("assetIDs:\ngot:  %v\nwant: %v", gWalletBuckets, wWalletBuckets)
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
				{id: "asset-id-0", label: "asset-0", agID: "ag-id-0", appID: "app-id-0"},
				{id: "asset-id-2", label: "asset-2", agID: "ag-id-1", appID: "app-id-0"},
			},
		},
		{
			[]string{"asset-id-1"},
			[]*actAsset{
				{id: "asset-id-1", label: "asset-1", agID: "ag-id-0", appID: "app-id-0"},
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

func TestGetActAddrs(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		addrIDs []string
		want    []*actAddr
	}{
		{
			[]string{"addr-id-0", "addr-id-1"},
			[]*actAddr{
				{id: "addr-id-0", address: "addr-0", isChange: false},
				{id: "addr-id-1", address: "addr-1", isChange: true},
			},
		},
		{
			[]string{"addr-id-2", "addr-id-4"},
			[]*actAddr{
				{id: "addr-id-2", address: "addr-2", isChange: false},
				{id: "addr-id-4", address: "addr-4", isChange: false},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.addrIDs)

		got, err := getActAddrs(ctx, ex.addrIDs)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("addrs:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestGetActBuckets(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		bucketIDs []string
		want      []*actBucket
	}{
		{
			[]string{"bucket-id-0", "bucket-id-2"},
			[]*actBucket{
				{id: "bucket-id-0", label: "bucket-0", walletID: "wallet-id-0", appID: "app-id-0"},
				{id: "bucket-id-2", label: "bucket-2", walletID: "wallet-id-1", appID: "app-id-0"},
			},
		},
		{
			[]string{"bucket-id-1"},
			[]*actBucket{
				{id: "bucket-id-1", label: "bucket-1", walletID: "wallet-id-0", appID: "app-id-0"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.bucketIDs)

		got, err := getActBuckets(ctx, ex.bucketIDs)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("buckets:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestCoalesceActivity(t *testing.T) {
	examples := []struct {
		ins, outs                   []*actUTXO
		visibleBuckets, changeAddrs []string
		want                        txRawActivity
	}{
		// Simple transfer from Alice's perspective
		{
			ins: []*actUTXO{
				{assetID: "gold", amount: 10, addrID: "alice-addr-id-0", bucketID: "alice-bucket-id-0"},
			},
			outs: []*actUTXO{
				{assetID: "gold", amount: 10, addrID: "bob-addr-id-0", bucketID: "bob-bucket-id-0"},
			},

			visibleBuckets: []string{"alice-bucket-id-0"},

			want: txRawActivity{
				insByA: map[string]map[string]int64{},
				insByB: map[string]map[string]int64{
					"alice-bucket-id-0": map[string]int64{"gold": 10},
				},
				outsByA: map[string]map[string]int64{
					"bob-addr-id-0": map[string]int64{"gold": 10},
				},
				outsByB: map[string]map[string]int64{},
			},
		},

		// Simple transfer from Bob's perspective
		{
			ins: []*actUTXO{
				{assetID: "gold", amount: 10, addrID: "alice-addr-id-0", bucketID: "alice-bucket-id-0"},
			},
			outs: []*actUTXO{
				{assetID: "gold", amount: 10, addrID: "bob-addr-id-0", bucketID: "bob-bucket-id-0"},
			},

			visibleBuckets: []string{"bob-bucket-id-0"},

			want: txRawActivity{
				insByA: map[string]map[string]int64{
					"alice-addr-id-0": map[string]int64{"gold": 10},
				},
				insByB:  map[string]map[string]int64{},
				outsByA: map[string]map[string]int64{},
				outsByB: map[string]map[string]int64{
					"bob-bucket-id-0": map[string]int64{"gold": 10},
				},
			},
		},

		// Trade from Alice's perspective
		{
			ins: []*actUTXO{
				{assetID: "gold", amount: 20, addrID: "alice-addr-id-0", bucketID: "alice-bucket-id-0"},
				{assetID: "silver", amount: 10, addrID: "bob-addr-id-0", bucketID: "bob-bucket-id-0"},
				{assetID: "silver", amount: 10, addrID: "bob-addr-id-1", bucketID: "bob-bucket-id-0"},
			},
			outs: []*actUTXO{
				{assetID: "silver", amount: 15, addrID: "alice-addr-id-1", bucketID: "alice-bucket-id-0"},
				{assetID: "gold", amount: 5, addrID: "bob-addr-id-2", bucketID: "bob-bucket-id-0"},

				{assetID: "gold", amount: 15, addrID: "alice-addr-id-2", bucketID: "alice-bucket-id-0"},
				{assetID: "silver", amount: 5, addrID: "bob-addr-id-3", bucketID: "bob-bucket-id-0"},
			},
			changeAddrs: []string{"alice-addr-id-2", "bob-addr-id-3"},

			visibleBuckets: []string{"alice-bucket-id-0"},

			want: txRawActivity{
				insByA: map[string]map[string]int64{
					"bob-addr-id-0": map[string]int64{"silver": 10},
					"bob-addr-id-1": map[string]int64{"silver": 10},
				},
				insByB: map[string]map[string]int64{
					"alice-bucket-id-0": map[string]int64{"gold": 5},
				},
				outsByA: map[string]map[string]int64{
					"bob-addr-id-2": map[string]int64{"gold": 5},
					"bob-addr-id-3": map[string]int64{"silver": 5},
				},
				outsByB: map[string]map[string]int64{
					"alice-bucket-id-0": map[string]int64{"silver": 15},
				},
			},
		},

		// Trade from Bob's perspective
		{
			ins: []*actUTXO{
				{assetID: "gold", amount: 20, addrID: "alice-addr-id-0", bucketID: "alice-bucket-id-0"},
				{assetID: "silver", amount: 10, addrID: "bob-addr-id-0", bucketID: "bob-bucket-id-0"},
				{assetID: "silver", amount: 10, addrID: "bob-addr-id-1", bucketID: "bob-bucket-id-0"},
			},
			outs: []*actUTXO{
				{assetID: "silver", amount: 15, addrID: "alice-addr-id-1", bucketID: "alice-bucket-id-0"},
				{assetID: "gold", amount: 5, addrID: "bob-addr-id-2", bucketID: "bob-bucket-id-0"},

				{assetID: "gold", amount: 15, addrID: "alice-addr-id-2", bucketID: "alice-bucket-id-0"},
				{assetID: "silver", amount: 5, addrID: "bob-addr-id-3", bucketID: "bob-bucket-id-0"},
			},
			changeAddrs: []string{"alice-addr-id-2", "bob-addr-id-3"},

			visibleBuckets: []string{"bob-bucket-id-0"},

			want: txRawActivity{
				insByA: map[string]map[string]int64{
					"alice-addr-id-0": map[string]int64{"gold": 20},
				},
				insByB: map[string]map[string]int64{
					"bob-bucket-id-0": map[string]int64{"silver": 15},
				},
				outsByA: map[string]map[string]int64{
					"alice-addr-id-1": map[string]int64{"silver": 15},
					"alice-addr-id-2": map[string]int64{"gold": 15},
				},
				outsByB: map[string]map[string]int64{
					"bob-bucket-id-0": map[string]int64{"gold": 5},
				},
			},
		},
	}

	for i, ex := range examples {
		t.Log("Example", i)

		got := coalesceActivity(ex.ins, ex.outs, ex.visibleBuckets, ex.changeAddrs)

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("coalesced activity:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestCreateActEntries(t *testing.T) {
	r := txRawActivity{
		insByA: map[string]map[string]int64{
			"addr-id-0": map[string]int64{
				"asset-id-0": 1,
				"asset-id-1": 2,
			},
			"addr-id-1": map[string]int64{
				"asset-id-2": 3,
				"asset-id-3": 4,
			},
		},
		insByB: map[string]map[string]int64{
			"bucket-id-0": map[string]int64{
				"asset-id-4": 5,
				"asset-id-5": 6,
			},
		},
		outsByA: map[string]map[string]int64{
			"addr-id-2": map[string]int64{
				"asset-id-5": 6,
				"asset-id-4": 5,
			},
		},
		outsByB: map[string]map[string]int64{
			"bucket-id-1": map[string]int64{
				"asset-id-3": 4,
				"asset-id-2": 3,
			},
			"bucket-id-2": map[string]int64{
				"asset-id-1": 2,
				"asset-id-0": 1,
			},
		},
	}

	addrs := map[string]string{
		"addr-id-0": "space-mountain",
		"addr-id-1": "small-world",
		"addr-id-2": "teacups",
	}

	assetLabels := map[string]string{
		"asset-id-0": "gold",
		"asset-id-1": "silver",
		"asset-id-2": "frankincense",
		"asset-id-3": "myrrh",
		"asset-id-4": "avocado",
		"asset-id-5": "mango",
	}

	bucketLabels := map[string]string{
		"bucket-id-0": "charlie",
		"bucket-id-1": "bob",
		"bucket-id-2": "alice",
	}

	gotIns, gotOuts := createActEntries(r, addrs, assetLabels, bucketLabels)

	wantIns := []actEntry{
		actEntry{BucketID: "bucket-id-0", BucketLabel: "charlie", AssetID: "asset-id-4", AssetLabel: "avocado", Amount: 5},
		actEntry{BucketID: "bucket-id-0", BucketLabel: "charlie", AssetID: "asset-id-5", AssetLabel: "mango", Amount: 6},
		actEntry{Address: "small-world", AssetID: "asset-id-2", AssetLabel: "frankincense", Amount: 3},
		actEntry{Address: "small-world", AssetID: "asset-id-3", AssetLabel: "myrrh", Amount: 4},
		actEntry{Address: "space-mountain", AssetID: "asset-id-0", AssetLabel: "gold", Amount: 1},
		actEntry{Address: "space-mountain", AssetID: "asset-id-1", AssetLabel: "silver", Amount: 2},
	}

	wantOuts := []actEntry{
		actEntry{BucketID: "bucket-id-2", BucketLabel: "alice", AssetID: "asset-id-0", AssetLabel: "gold", Amount: 1},
		actEntry{BucketID: "bucket-id-2", BucketLabel: "alice", AssetID: "asset-id-1", AssetLabel: "silver", Amount: 2},
		actEntry{BucketID: "bucket-id-1", BucketLabel: "bob", AssetID: "asset-id-2", AssetLabel: "frankincense", Amount: 3},
		actEntry{BucketID: "bucket-id-1", BucketLabel: "bob", AssetID: "asset-id-3", AssetLabel: "myrrh", Amount: 4},
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

func TestWriteWalletActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	buckets := []string{"bucket-id-0", "bucket-id-1"}
	bucketSet := make(map[string]bool)
	for _, b := range buckets {
		bucketSet[b] = true
	}

	err := writeWalletActivity(
		ctx,
		"wallet-id-0", "tx-hash", []byte(`{"transaction_id":"dummy"}`),
		buckets,
	)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	gotAct, err := getTestActivity(ctx, "tx-hash", false)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	wantAct := map[string]actItem{"wallet-id-0": actItem{TxHash: "dummy"}}
	if !reflect.DeepEqual(gotAct, wantAct) {
		t.Errorf("activity rows:\ngot:  %v\nwant: %v", gotAct, wantAct)
	}

	gotBuckets, err := getTestActivityBuckets(ctx, "tx-hash")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	wantBuckets := []string{"bucket-id-0", "bucket-id-1"}
	if !reflect.DeepEqual(gotBuckets, wantBuckets) {
		t.Errorf("buckets:\ngot:  %v\nwant: %v", gotBuckets, wantBuckets)
	}
}

func TestWriteIssuanceActivity(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, writeActivityFix)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	err := writeIssuanceActivity(
		ctx,
		&actAsset{id: "asset-id-0", agID: "ag-id-0"},
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

	wantAct := map[string]actItem{"ag-id-0": actItem{TxHash: "dummy"}}
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
	relationID := "wallet_id"
	table := "activity"
	if issuance {
		relationID = "asset_group_id"
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

func getTestActivityBuckets(ctx context.Context, txHash string) ([]string, error) {
	q := `
		SELECT array_agg(bucket_id ORDER BY bucket_id)
		FROM activity_buckets ab
		JOIN activity a ON ab.activity_id = a.id
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

func mustHash32FromStr(s string) wire.Hash32 {
	h, err := wire.NewHash32FromStr(s)
	if err != nil {
		panic(err)
	}
	return *h
}

func stringsToRawJSON(strs ...string) []*json.RawMessage {
	var res []*json.RawMessage
	for _, s := range strs {
		b := json.RawMessage([]byte(s))
		res = append(res, &b)
	}
	return res
}
