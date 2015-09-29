package appdb

import (
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/errors"
	"chain/fedchain-sandbox/wire"
)

const sampleActivityFixture = `
		INSERT INTO wallets (id, application_id, label, current_rotation, key_index)
			VALUES('w0', 'app-id-0', '', 'c0', 0);
		INSERT INTO activity (id, wallet_id, data, txid)
			VALUES('act0', 'w0', '{"outputs":"boop"}', 'tx0');
	`

const sampleActivityWalletFixture = `
		INSERT INTO wallets (id, application_id, label, current_rotation, key_index)
			VALUES
				('w0', 'app-id-0', '', 'c0', 0),
				('w1', 'app-id-0', '', 'c0', 0);
		INSERT INTO buckets (id, wallet_id, key_index, label)
			VALUES
				('b0', 'w0', 0, 'account zero'),
				('b1', 'w0', 1, 'account one'),
				('b3', 'w1', 2, 'account three');
`

const sampleAssetActivityFixture = `
	INSERT INTO asset_groups (id, application_id, key_index, label, keyset)
		VALUES ('ag','app-id-0', 0, 'whatever', '{}');
	INSERT INTO assets (id, asset_group_id, key_index, redeem_script, label)
		VALUES ('a0', 'ag', 0, '', 'asset zero');
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

// The txid values in the fixture data are not arbitrary.
// 0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098 is from the btcd/wire tests.
// 3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df is the hash of the tx generated in this test file.
const createActivityItemFixture = `
	INSERT INTO addresses
			(id, wallet_id, bucket_id, keyset, key_index, address, is_change, redeem_script, pk_script)
		VALUES
			('addr0', 'w0', 'b0', '{}', 2, 'aaac', false, '', ''),
			('addr1', 'w0', 'b1', '{}', 0, 'aaaa', false, '', '');

	INSERT INTO utxos
			(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES
			('0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 0, 'a0', 100, 'addr0', 'b0', 'w0'),
			('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 1, 'a0', 50, 'addr1', 'b1', 'w0');
`

type activityRes struct {
	id        string
	wallet_id string
	data      []byte
}

func TestCreateActivityItem(t *testing.T) {
	cases := []struct {
		fixture string
		want    []activityRes
	}{
		{
			fixture: `
				INSERT INTO addresses
					(id, wallet_id, bucket_id, keyset, key_index, address, is_change, redeem_script, pk_script)
				VALUES
					('addr2', 'w1', 'b3', '{}', 1, 'aaab', false, '', '');

				INSERT INTO utxos
					(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
				VALUES
					('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 2, 'a0', 50, 'addr2', 'b3', 'w1');
			`,
			want: []activityRes{
				{
					wallet_id: "w0",
					data:      []byte(`{"inputs":[{"account_id":"b0","account_label":"account zero","amount":100,"asset_id":"a0","asset_label":"asset zero"}],"outputs":[{"address":"addr2","amount":50,"asset_id":"a0","asset_label":"asset zero"},{"account_id":"b1","account_label":"account one","amount":50,"asset_id":"a0","asset_label":"asset zero"}],"transaction_id":"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df","transaction_time":"2015-09-17T12:50:53.427092Z"}`),
				},
				{
					wallet_id: "w1",
					data:      []byte(`{"inputs":[{"address":"addr0","amount":100,"asset_id":"a0","asset_label":"asset zero"}],"outputs":[{"address":"addr1","amount":50,"asset_id":"a0","asset_label":"asset zero"},{"account_id":"b3","account_label":"account three","amount":50,"asset_id":"a0","asset_label":"asset zero"}],"transaction_id":"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df","transaction_time":"2015-09-17T12:50:53.427092Z"}`),
				},
			},
		},
		{
			fixture: `
					INSERT INTO addresses
						(id, wallet_id, bucket_id, keyset, key_index, address, is_change, redeem_script, pk_script)
					VALUES
						('addr2', 'w1', 'b3', '{}', 1, 'aaab', true, '', '');

					INSERT INTO utxos
						(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
					VALUES
						('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 2, 'a0', 50, 'addr2', 'b0', 'w0'); -- CHANGE OUTPUT
				`,
			want: []activityRes{
				{
					wallet_id: "w0",
					data:      []byte(`{"inputs":[{"account_id":"b0","account_label":"account zero","amount":50,"asset_id":"a0","asset_label":"asset zero"}],"outputs":[{"account_id":"b1","account_label":"account one","amount":50,"asset_id":"a0","asset_label":"asset zero"}],"transaction_id":"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df","transaction_time":"2015-09-17T12:50:53.427092Z"}`),
				},
			},
		},
	}

	for _, test := range cases {
		dbtx := pgtest.TxWithSQL(t,
			sampleAppFixture,
			sampleActivityWalletFixture,
			createActivityItemFixture,
			sampleAssetActivityFixture,
			test.fixture)
		ctx := pg.NewContext(context.Background(), dbtx)
		defer dbtx.Rollback()

		hashStr := "0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098"
		inHash, err := wire.NewHash32FromStr(hashStr)
		if err != nil {
			t.Fatalf("Unexpected err creating fixture hash %v", err)
		}

		txTime, err := time.Parse(time.RFC3339, "2015-09-17T12:50:53.427092Z")
		if err != nil {
			t.Fatalf("unexpected err parsing time %v", err)
		}

		tx := wire.NewMsgTx()
		tx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: wire.OutPoint{
				Hash:  *inHash,
				Index: 0,
			},
			Sequence: 4294967295,
		})

		err = CreateActivityItems(ctx, tx, txTime)
		if err != nil {
			t.Fatalf("Unexpected err %v", err)
		}

		const q = `SELECT wallet_id, data FROM activity ORDER BY wallet_id`
		rows, err := pg.FromContext(ctx).Query(q)
		if err != nil {
			t.Fatalf("Unexpected err %v", err)
		}
		defer rows.Close()

		var (
			wallet_id string
			data      []byte
		)

		var res []activityRes
		for rows.Next() {
			err = rows.Scan(&wallet_id, &data)
			r := activityRes{
				wallet_id: wallet_id,
				data:      data,
			}
			res = append(res, r)
		}

		if !reflect.DeepEqual(test.want, res) {
			t.Fatalf("activity:\ngot:  %v\nwant: %v", string(res[0].data), string(test.want[0].data))
		}

		dbtx.Rollback()
	}
}

func TestGenerateActivityFromUtxo(t *testing.T) {
	// The txid values in the fixture data are not arbitrary.
	// 0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098 is from the btcd/wire tests.
	// 3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df is the hash of the tx generated in this test file.
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, sampleActivityWalletFixture, sampleAssetActivityFixture, `
		INSERT INTO addresses
			(id, wallet_id, bucket_id, keyset, key_index, address, is_change, redeem_script, pk_script)
		VALUES
			('addr0', 'w0', 'b0', '{}', 2, 'aaac', false, '', ''),
			('addr1', 'w0', 'b1', '{}', 0, 'aaaa', false, '', ''),
			('addr2', 'w1', 'b3', '{}', 1, 'aaab', false, '', '');

		INSERT INTO utxos
			(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES
			('0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 0, 'a0', 100, 'addr0', 'b0', 'w0'),
			('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 1, 'a0', 50, 'addr1', 'b1', 'w0'),
			('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 2, 'a0', 50, 'addr2', 'b3', 'w1'),
			('0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 3, 'a0', 100, 'addr0', 'b0', 'w0');
			-- this last utxo shouldn't get selected because it isn't included in the transaction below
	`)
	ctx := pg.NewContext(context.Background(), dbtx)
	defer dbtx.Rollback()

	hashStr := "0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098"
	inHash, err := wire.NewHash32FromStr(hashStr)
	if err != nil {
		t.Fatalf("Unexpected err creating fixture hash %v", err)
	}

	txTime, err := time.Parse(time.RFC3339, "2015-09-17T12:50:53.427092Z")
	if err != nil {
		t.Fatalf("unexpected err parsing time %v", err)
	}

	tx := wire.NewMsgTx()
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  *inHash,
			Index: 0,
		},
		Sequence: 4294967295,
	})

	err = CreateActivityItems(ctx, tx, txTime)
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}

	// No change in this transaction.
	addrIsChange := map[string]bool{}

	item, err := generateActivityItem(ctx, tx, "w0", addrIsChange, txTime)
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}

	if item.walletID != "w0" {
		t.Fatalf("Want walletID=w0 got=%s", item.walletID)
	}

	if item.txid != "3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df" {
		t.Fatalf("Want txid=3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df, got=%s", item.txid)
	}

	wantData := `{"inputs":[{"account_id":"b0","account_label":"account zero","amount":100,"asset_id":"a0","asset_label":"asset zero"}],"outputs":[{"address":"addr2","amount":50,"asset_id":"a0","asset_label":"asset zero"},{"account_id":"b1","account_label":"account one","amount":50,"asset_id":"a0","asset_label":"asset zero"}],"transaction_id":"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df","transaction_time":"2015-09-17T12:50:53.427092Z"}`
	if string(item.data) != wantData {
		t.Fatalf("activity:\ngot:  %v\nwant: %v", string(item.data), wantData)
	}

}
