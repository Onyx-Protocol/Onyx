package appdb

import (
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"testing"
	"time"

	"golang.org/x/net/context"

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
		INSERT INTO buckets (id, wallet_id, key_index)
			VALUES
				('b0', 'w0', 0),
				('b1', 'w0', 1),
				('b3', 'w1', 2);
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
}

func TestWriteActivity(t *testing.T) {
	// The txid values in the fixture data are not arbitrary.
	// 0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098 is from the btcd/wire tests.
	// 3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df is the hash of the tx generated in this test file.
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, sampleActivityWalletFixture, `
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
			('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 2, 'a0', 50, 'addr2', 'b3', 'w1');
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

	const q = `SELECT id, wallet_id, data FROM activity ORDER BY wallet_id`
	rows, err := pg.FromContext(ctx).Query(q)
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}
	defer rows.Close()

	type activityRes struct {
		id        string
		wallet_id string
		data      []byte
	}

	var (
		id        string
		wallet_id string
		data      []byte
	)

	var res []*activityRes
	for rows.Next() {
		err = rows.Scan(&id, &wallet_id, &data)
		r := &activityRes{
			id:        id,
			wallet_id: wallet_id,
			data:      data,
		}
		res = append(res, r)
	}

	if res[0].wallet_id != "w0" {
		t.Fatalf("want w0 got=%v", res[0].wallet_id)
	}
	if res[1].wallet_id != "w1" {
		t.Fatalf("want w1 got=%v", res[1].wallet_id)
	}

	// The notable distinction in the data is the way that addresses and buckets are handled.
	// For activity in a given wallet, we should see buckets for inputs and outputs that came
	// from that wallet, but addresses otherwise.
	wantData0 := `{"inputs":[{"amount":100,"asset_id":"a0","bucket_id":"b0"}],"outputs":[{"address":"addr2","amount":50,"asset_id":"a0"},{"amount":50,"asset_id":"a0","bucket_id":"b1"}],"transaction_time":"2015-09-17T12:50:53.427092Z","txid":"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df"}`
	if string(res[0].data) != wantData0 {
		t.Fatalf("want=%s got=%s", wantData0, string(res[0].data))
	}

	wantData1 := `{"inputs":[{"address":"addr0","amount":100,"asset_id":"a0"}],"outputs":[{"address":"addr1","amount":50,"asset_id":"a0"},{"amount":50,"asset_id":"a0","bucket_id":"b3"}],"transaction_time":"2015-09-17T12:50:53.427092Z","txid":"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df"}`
	if string(res[1].data) != wantData1 {
		t.Fatalf("want=%s got=%s", wantData1, string(res[1].data))
	}
}

func TestWriteActivityWithChangeOutputs(t *testing.T) {
	// As in the test above, the txid values in the fixture data are not arbitrary.
	// 0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098 is from the btcd/wire tests.
	// 3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df is the hash of the tx generated in this test file.
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, sampleActivityWalletFixture, `
		INSERT INTO addresses
			(id, wallet_id, bucket_id, keyset, key_index, address, is_change, redeem_script, pk_script)
		VALUES
			('addr0', 'w0', 'b0', '{}', 2, 'aaac', false, '', ''),
			('addr1', 'w0', 'b1', '{}', 0, 'aaaa', false, '', ''),
			('addr2', 'w1', 'b3', '{}', 1, 'aaab', true, '', '');

		INSERT INTO utxos
			(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES
			('0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098', 0, 'a0', 100, 'addr0', 'b0', 'w0'),
			('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 1, 'a0', 50, 'addr1', 'b1', 'w0'),
			('3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df', 2, 'a0', 50, 'addr2', 'b0', 'w0'); -- CHANGE OUTPUT
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

	const q = `SELECT id, wallet_id, data FROM activity ORDER BY wallet_id`
	rows, err := pg.FromContext(ctx).Query(q)
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}
	defer rows.Close()

	type activityRes struct {
		id        string
		wallet_id string
		data      []byte
	}

	var (
		id        string
		wallet_id string
		data      []byte
	)

	var res []*activityRes
	for rows.Next() {
		err = rows.Scan(&id, &wallet_id, &data)
		r := &activityRes{
			id:        id,
			wallet_id: wallet_id,
			data:      data,
		}
		res = append(res, r)
	}

	if len(res) != 1 {
		t.Fatalf("Should only be one activity response, got %v", len(res))
	}

	if res[0].wallet_id != "w0" {
		t.Fatalf("want w0 got=%v", res[0].wallet_id)
	}

	// The change output should not appear.
	wantData0 := `{"inputs":[{"amount":50,"asset_id":"a0","bucket_id":"b0"}],"outputs":[{"amount":50,"asset_id":"a0","bucket_id":"b1"}],"transaction_time":"2015-09-17T12:50:53.427092Z","txid":"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df"}`
	if string(res[0].data) != wantData0 {
		t.Fatalf("want=%s got=%s", wantData0, string(res[0].data))
	}
}

func TestWriteActivityFromUtxo(t *testing.T) {
	// The txid values in the fixture data are not arbitrary.
	// 0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098 is from the btcd/wire tests.
	// 3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df is the hash of the tx generated in this test file.
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, sampleActivityWalletFixture, `
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

	const q = `SELECT id, wallet_id, data FROM activity ORDER BY wallet_id`
	rows, err := pg.FromContext(ctx).Query(q)
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}
	defer rows.Close()

	type activityRes struct {
		id        string
		wallet_id string
		data      []byte
	}

	var (
		id        string
		wallet_id string
		data      []byte
	)

	var res []*activityRes
	for rows.Next() {
		err = rows.Scan(&id, &wallet_id, &data)
		r := &activityRes{
			id:        id,
			wallet_id: wallet_id,
			data:      data,
		}
		res = append(res, r)
	}

	if len(res) != 2 {
		t.Fatalf("wanted 2 activity items got=%d", len(res))
	}

	if res[0].wallet_id != "w0" {
		t.Fatalf("want w0 got=%v", res[0].wallet_id)
	}
	if res[1].wallet_id != "w1" {
		t.Fatalf("want w1 got=%v", res[1].wallet_id)
	}

	// The notable distinction in the data is the way that addresses and buckets are handled.
	// For activity in a given wallet, we should see buckets for inputs and outputs that came
	// from that wallet, but addresses otherwise.
	wantData0 := `{"inputs":[{"amount":100,"asset_id":"a0","bucket_id":"b0"}],"outputs":[{"address":"addr2","amount":50,"asset_id":"a0"},{"amount":50,"asset_id":"a0","bucket_id":"b1"}],"transaction_time":"2015-09-17T12:50:53.427092Z","txid":"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df"}`
	if string(res[0].data) != wantData0 {
		t.Fatalf("want=%s got=%s", wantData0, string(res[0].data))
	}

	wantData1 := `{"inputs":[{"address":"addr0","amount":100,"asset_id":"a0"}],"outputs":[{"address":"addr1","amount":50,"asset_id":"a0"},{"amount":50,"asset_id":"a0","bucket_id":"b3"}],"transaction_time":"2015-09-17T12:50:53.427092Z","txid":"3924f077fedeb24248f9e63532433473710a4df88df4805425a16598dd3f58df"}`
	if string(res[1].data) != wantData1 {
		t.Fatalf("want=%s got=%s", wantData1, string(res[1].data))
	}
}
