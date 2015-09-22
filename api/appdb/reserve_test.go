package appdb

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/errors"
	"chain/fedchain-sandbox/wire"
)

func TestReserveUTXOs(t *testing.T) {
	const outs = `
		INSERT INTO utxos
		(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES
			('b8eb9723231326795e8022269ad88603761ca65aa397988f0a0909f7702f2e45', 0, 'a1', 1, 'a1', 'b1', 'w1'),
			('b8eb9723231326795e8022269ad88603761ca65aa397988f0a0909f7702f2e45', 1, 'a1', 1, 'a2', 'b1', 'w1');
	`
	cases := []struct {
		askAmt  int64
		wantErr error
		want    int
	}{{
		askAmt:  5000,
		wantErr: ErrInsufficientFunds,
	}, {
		askAmt: 1,
		want:   1,
	}}

	for _, c := range cases {
		dbtx := pgtest.TxWithSQL(t, outs)
		ctx := pg.NewContext(context.Background(), dbtx)
		got, _, err := ReserveUTXOs(ctx, "a1", "b1", c.askAmt, time.Minute)

		if err != c.wantErr {
			t.Errorf("got err = %q want %q", err, c.wantErr)
			dbtx.Rollback()
			continue
		}

		if len(got) != c.want {
			t.Errorf("got len(outs) = %d want %d", len(got), c.want)
		}
		dbtx.Rollback()
	}
}

func TestReserveTxUTXOs(t *testing.T) {
	const outs = `
		INSERT INTO utxos
		(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES
			('b8eb9723231326795e8022269ad88603761ca65aa397988f0a0909f7702f2e45', 0, 'a1', 1, 'a1', 'b1', 'w1'),
			('b8eb9723231326795e8022269ad88603761ca65aa397988f0a0909f7702f2e45', 1, 'a1', 1, 'a2', 'b1', 'w1');
	`
	hash, _ := wire.NewHash32FromStr("b8eb9723231326795e8022269ad88603761ca65aa397988f0a0909f7702f2e45")
	cases := []struct {
		askAmt  int64
		wantErr error
		want    []*UTXO
	}{{
		askAmt:  5000,
		wantErr: ErrInsufficientFunds,
	}, {
		askAmt: 1,
		want: []*UTXO{{
			OutPoint:  wire.NewOutPoint(hash, 0),
			Amount:    1,
			AddressID: "a1",
		}},
	}}

	for _, c := range cases {
		dbtx := pgtest.TxWithSQL(t, outs)
		ctx := pg.NewContext(context.Background(), dbtx)
		got, _, err := ReserveTxUTXOs(ctx,
			"a1", "b1", "b8eb9723231326795e8022269ad88603761ca65aa397988f0a0909f7702f2e45", c.askAmt, time.Minute)

		if err != c.wantErr {
			t.Errorf("got err = %q want %q", err, c.wantErr)
			dbtx.Rollback()
			continue
		}

		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("got outs = %v want %v", got, c.want)
		}
		dbtx.Rollback()
	}
}

func TestReserveSQL(t *testing.T) {
	var threeUTXOsFixture = `
		INSERT INTO utxos
		(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES
			('t1', 0, 'a1', 1, 'a1', 'b1', 'w1'),
			('t2', 0, 'a1', 1, 'a2', 'b1', 'w1'),
			('t3', 0, 'a1', 1, 'a3', 'b1', 'w1');
	`

	type want struct {
		Txid       string
		Index, Amt int
		AddressID  string
	}
	tests := []struct {
		description  string
		fixture      string
		askAmt       int
		wantErr      string
		want         int
		wantReserved int
	}{
		{
			description:  "test reserves minimum needed",
			fixture:      threeUTXOsFixture,
			askAmt:       2,
			want:         2,
			wantReserved: 2,
		},
		{
			description: "test returns error if minimum is not met",
			fixture:     threeUTXOsFixture,
			askAmt:      4,
			wantErr:     "insufficient funds",
		},
		{
			description: "test does not return already reserved utxos",
			fixture: `
				INSERT INTO utxos
				(txid, index, asset_id, amount, address_id, bucket_id, wallet_id, reserved_until)
				VALUES
					('t1', 0, 'a1', 1, 'a1', 'b1', 'w1', now()+'60s'::interval),
					('t2', 0, 'a1', 1, 'a2', 'b1', 'w1', now()-'1s'::interval);
			`,
			askAmt:       1,
			want:         1,
			wantReserved: 2,
		},
	}

	for _, test := range tests {
		t.Log(test.description)
		dbtx := pgtest.TxWithSQL(t, test.fixture)

		rows, err := dbtx.Query(`SELECT * FROM reserve_utxos('a1', 'b1', $1, '60s'::interval)`, test.askAmt)
		if pqErr, ok := (err).(*pq.Error); ok {
			if !strings.Contains(pqErr.Message, test.wantErr) {
				t.Errorf("got error = %q want %q", pqErr.Message, test.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			dbtx.Rollback()
			continue
		}

		var got []want
		err = sql.Collect(rows, &got)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			dbtx.Rollback()
			continue
		}

		if len(got) != test.want {
			t.Errorf("got len reserve_utxos(%d) = %d want %d", test.askAmt, len(got), test.want)
		}

		const onlyReservedQ = `
			SELECT COUNT(*) FROM utxos
			WHERE reserved_until > now()
		`

		var reservedCnt int
		err = dbtx.QueryRow(onlyReservedQ).Scan(&reservedCnt)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			dbtx.Rollback()
			continue
		}

		if reservedCnt != test.wantReserved {
			t.Errorf("got utxos reserved = %d want %d", reservedCnt, test.wantReserved)
		}

		dbtx.Rollback()
	}
}

func TestReserveTxSQL(t *testing.T) {
	type utxo struct {
		Txid       string
		Index, Amt int
		AddressID  string
	}

	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO utxos
		(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES
			('t1', 0, 'a1', 1, 'a1', 'b1', 'w1'),
			('t2', 0, 'a1', 1, 'a2', 'b1', 'w1'),
			('t1', 1, 'a1', 1, 'a3', 'b1', 'w1');
	`)
	defer dbtx.Rollback()

	rows, err := dbtx.Query(`SELECT * FROM reserve_tx_utxos('a1', 'b1', 't1', 2, '60s'::interval)`)
	if err != nil {
		t.Fatal(err)
	}
	var got []utxo
	err = sql.Collect(rows, &got)
	if err != nil {
		t.Fatal(err)
	}

	want := []utxo{{"t1", 0, 1, "a1"}, {"t1", 1, 1, "a3"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got utxos = %+v want %+v", got, want)
	}
}

func TestCancelReservations(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO utxos
		(txid, index, asset_id, amount, address_id, bucket_id, wallet_id, reserved_until)
		VALUES
			('0000000000000000000000000000000000000000000000000000000000000001', 0, 'a1', 1, 'a1', 'b1', 'w1', NOW()+'1h'::interval),
			('0000000000000000000000000000000000000000000000000000000000000002', 0, 'a1', 1, 'a2', 'b1', 'w1', NOW()+'1h'::interval),
			('0000000000000000000000000000000000000000000000000000000000000003', 0, 'a1', 1, 'a2', 'b1', 'w1', NOW()+'1h'::interval);
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	outpoints := []wire.OutPoint{{
		Hash:  wire.Hash32([32]byte{1}),
		Index: 0,
	}, {
		Hash:  wire.Hash32([32]byte{2}),
		Index: 0,
	}}

	err := CancelReservations(ctx, outpoints)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	const q = `SELECT COUNT(*) FROM utxos WHERE reserved_until <= NOW()`
	var cnt int
	err = dbtx.QueryRow(q).Scan(&cnt)
	if err != nil {
		t.Fatal(err)
	}

	if cnt != 2 {
		t.Errorf("got free utxos=%d want %d", cnt, 2)
	}
}
