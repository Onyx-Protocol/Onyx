package appdb

import (
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"testing"

	"golang.org/x/net/context"
)

const sampleActivityFixture = `
		INSERT INTO wallets (id, application_id, label, current_rotation, key_index)
			VALUES('w0', 'app-id-0', '', 'c0', 0);
		INSERT INTO activity (id, wallet_id, data, txid)
			VALUES('act0', 'w0', '{"outputs":"boop"}', 'tx0')
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
