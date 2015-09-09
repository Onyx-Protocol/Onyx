package appdb

import (
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"reflect"
	"strings"
	"testing"

	"github.com/lib/pq"
)

func TestReserveSQL(t *testing.T) {
	var threeUTXOsFixture = `
		INSERT INTO utxos
		(txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
		VALUES
			('t1', 0, 'a1', 1, 'r1', 'b1', 'w1'),
			('t2', 0, 'a1', 1, 'r2', 'b1', 'w1'),
			('t3', 0, 'a1', 1, 'r3', 'b1', 'w1');
	`

	type want struct {
		Txid       string
		Index, Amt int
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
			want:         []want{{"t1", 0, 1}, {"t2", 0, 1}},
			wantReserved: []want{{"t1", 0, 1}, {"t2", 0, 1}},
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
					('t1', 0, 'a1', 1, 'r1', 'b1', 'w1', now()),
					('t2', 0, 'a1', 1, 'r2', 'b1', 'w1', now()-'61s'::interval);
			`,
			askAmt:       1,
			want:         []want{{"t2", 0, 1}},
			wantReserved: []want{{"t1", 0, 1}, {"t2", 0, 1}},
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
			SELECT txid, index, amount FROM utxos
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
