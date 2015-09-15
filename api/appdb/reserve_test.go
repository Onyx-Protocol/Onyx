package appdb

import (
	"reflect"
	"strings"
	"testing"

	"github.com/lib/pq"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
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
		got, _, err := ReserveUTXOs(ctx, "a1", "b1", c.askAmt)

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
			"a1", "b1", "b8eb9723231326795e8022269ad88603761ca65aa397988f0a0909f7702f2e45", c.askAmt)

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
		want         []want
		wantReserved []want
	}{
		{
			description:  "test reserves minimum needed",
			fixture:      threeUTXOsFixture,
			askAmt:       2,
			want:         []want{{"t1", 0, 1, "a1"}, {"t2", 0, 1, "a2"}},
			wantReserved: []want{{"t1", 0, 1, "a1"}, {"t2", 0, 1, "a2"}},
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
				(txid, index, asset_id, amount, address_id, bucket_id, wallet_id, reserved_at)
				VALUES
					('t1', 0, 'a1', 1, 'a1', 'b1', 'w1', now()),
					('t2', 0, 'a1', 1, 'a2', 'b1', 'w1', now()-'61s'::interval);
			`,
			askAmt:       1,
			want:         []want{{"t2", 0, 1, "a2"}},
			wantReserved: []want{{"t1", 0, 1, "a1"}, {"t2", 0, 1, "a2"}},
		},
	}

	for _, test := range tests {
		t.Log(test.description)
		dbtx := pgtest.TxWithSQL(t, test.fixture)

		rows, err := dbtx.Query(`SELECT * FROM reserve_utxos('a1', 'b1', $1)`, test.askAmt)
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

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got reserve_utxos(%d) = %+v want %+v", test.askAmt, got, test.want)
		}

		const onlyReservedQ = `
			SELECT txid, index, amount, address_id FROM utxos
			WHERE reserved_at > now()-'60s'::interval
			ORDER BY address_id ASC
		`

		rows, err = dbtx.Query(onlyReservedQ)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			dbtx.Rollback()
			continue
		}

		got = nil
		err = sql.Collect(rows, &got)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			dbtx.Rollback()
			continue
		}

		if !reflect.DeepEqual(got, test.wantReserved) {
			t.Errorf("got utxos reserved = %+v want %+v", got, test.wantReserved)
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

	rows, err := dbtx.Query(`SELECT * FROM reserve_tx_utxos('a1', 'b1', 't1', 2)`)
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
