package main

import (
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"reflect"
	"testing"
)

func TestReserve(t *testing.T) {
	type want struct {
		Txid       string
		Index, Amt int
	}
	tests := []struct {
		description string
		fixture     string
		askAmt      int
		want        []want
	}{
		{
			description: "test reserves minimum needed",
			fixture: `
				INSERT INTO outputs
				(txid, index, asset_id, amount, receiver_id, bucket_id, wallet_id)
				VALUES
					('t1', 0, 'a1', 1, 'r1', 'b1', 'w1'),
					('t2', 0, 'a1', 1, 'r2', 'b1', 'w1'),
					('t3', 0, 'a1', 1, 'r3', 'b1', 'w1');
			`,
			askAmt: 2,
			want:   []want{{"t1", 0, 1}, {"t2", 0, 1}},
		},
	}

	for _, test := range tests {
		t.Log(test.description)
		pgtest.LoadSQL(t, test.fixture)

		rows, err := db.Query(`SELECT * FROM reserve_outputs('a1', 'b1', $1)`, test.askAmt)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}

		var got []want
		err = sql.Collect(rows, &got)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got reserve_outputs(%d) = %+v want %+v", test.askAmt, got, test.want)
		}
	}
}
