package query

import (
	"context"
	"testing"
	"time"

	"chain/core/query/filter"
	"chain/database/pg/pgtest"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/testutil"
)

func TestDecodeOutputsAfter(t *testing.T) {
	testCases := []struct {
		str string
		cur OutputsAfter
	}{
		{str: "1:1:0", cur: OutputsAfter{lastBlockHeight: 1, lastTxPos: 1}},
		{str: "1:2:3", cur: OutputsAfter{lastBlockHeight: 1, lastTxPos: 2, lastIndex: 3}},
		{str: "10:1:0", cur: OutputsAfter{lastBlockHeight: 10, lastTxPos: 1}},
		{str: "15:15:15", cur: OutputsAfter{lastBlockHeight: 15, lastTxPos: 15, lastIndex: 15}},
		{str: "49153:51966:51829", cur: OutputsAfter{lastBlockHeight: 49153, lastTxPos: 51966, lastIndex: 51829}},
		{str: "9223372036854775807:4294967295:2147483647", cur: defaultOutputsAfter},
	}

	for _, tc := range testCases {
		decoded, err := DecodeOutputsAfter(tc.str)
		if err != nil {
			t.Error(err)
		}
		if !testutil.DeepEqual(decoded, &tc.cur) {
			t.Errorf("got %#v, want %#v", decoded, &tc.cur)
		}
		if decoded.String() != tc.str {
			t.Errorf("re-encode: got %s, want %s", decoded.String(), tc.str)
		}
	}
}

func TestOutputsAfter(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		INSERT INTO annotated_outputs (block_height, tx_pos, output_index, tx_hash, output_id, timespan,
			type, purpose, asset_id, asset_alias, asset_definition, asset_local, asset_tags, amount, control_program, reference_data, local)
		VALUES
		(1, 0, 0, 'ab', 'o1', int8range(1, 100), 'control', 'receive', E'\\xDEADBEEF', 'a', '{}'::jsonb, true, '{}'::jsonb, 10, E'\\xDEADBEEF', '{}'::jsonb, true),
		(1, 1, 0, 'cd', 'o2', int8range(1, 100), 'control', 'receive', E'\\xDEADBEEF', 'a', '{}'::jsonb, true, '{}'::jsonb, 10, E'\\xDEADBEEF', '{}'::jsonb, true),
		(1, 1, 1, 'cd', 'o3', int8range(1, 100), 'control', 'receive', E'\\xDEADBEEF', 'a', '{}'::jsonb, true, '{}'::jsonb, 10, E'\\xDEADBEEF', '{}'::jsonb, true),
		(2, 0, 0, 'ef', 'o4', int8range(10, 50), 'control', 'receive', E'\\xDEADBEEF', 'a', '{}'::jsonb, true, '{}'::jsonb, 10, E'\\xDEADBEEF', '{}'::jsonb, true);
	`)
	if err != nil {
		t.Fatal(err)
	}

	const q = `asset_id = 'deadbeef'`
	indexer := NewIndexer(db, &protocol.Chain{}, nil)
	results, after, err := indexer.Outputs(ctx, q, nil, 25, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
	if after.String() != "1:1:1" {
		t.Errorf("got after=%q want 1:1:1", after.String())
	}

	results, after, err = indexer.Outputs(ctx, q, nil, 25, after, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
	if after.String() != "1:0:0" {
		t.Errorf("got after=%q want 1:0:0", after.String())
	}
}

func TestConstructOutputsQuery(t *testing.T) {
	now := time.Unix(233400000, 0)
	nowMillis := bc.Millis(now)

	testCases := []struct {
		filter     string
		values     []interface{}
		after      *OutputsAfter
		wantQuery  string
		wantValues []interface{}
	}{
		{
			// empty filter
			wantQuery:  `SELECT block_height, tx_pos, output_index, tx_hash, output_id, type, purpose, asset_id, asset_alias, asset_definition, asset_tags, asset_local, amount, account_id, account_alias, account_tags, control_program, reference_data, local FROM "annotated_outputs" AS out WHERE timespan @> $1::int8 ORDER BY block_height DESC, tx_pos DESC, output_index DESC LIMIT 10`,
			wantValues: []interface{}{nowMillis},
		},
		{
			filter:     "asset_id = $1 AND account_id = 'abc'",
			values:     []interface{}{"foo"},
			wantQuery:  `SELECT block_height, tx_pos, output_index, tx_hash, output_id, type, purpose, asset_id, asset_alias, asset_definition, asset_tags, asset_local, amount, account_id, account_alias, account_tags, control_program, reference_data, local FROM "annotated_outputs" AS out WHERE (encode(out."asset_id", 'hex') = $1 AND out."account_id" = 'abc') AND timespan @> $2::int8 ORDER BY block_height DESC, tx_pos DESC, output_index DESC LIMIT 10`,
			wantValues: []interface{}{`foo`, nowMillis},
		},
		{
			filter: "asset_id = $1 AND account_id = 'abc'",
			values: []interface{}{"foo"},
			after: &OutputsAfter{
				lastBlockHeight: 15,
				lastTxPos:       17,
				lastIndex:       19,
			},
			wantQuery:  `SELECT block_height, tx_pos, output_index, tx_hash, output_id, type, purpose, asset_id, asset_alias, asset_definition, asset_tags, asset_local, amount, account_id, account_alias, account_tags, control_program, reference_data, local FROM "annotated_outputs" AS out WHERE (encode(out."asset_id", 'hex') = $1 AND out."account_id" = 'abc') AND timespan @> $2::int8 AND (block_height, tx_pos, output_index) < ($3, $4, $5) ORDER BY block_height DESC, tx_pos DESC, output_index DESC LIMIT 10`,
			wantValues: []interface{}{`foo`, nowMillis, uint64(15), uint32(17), 19},
		},
	}

	for i, tc := range testCases {
		f, err := filter.Parse(tc.filter, outputsTable, tc.values)
		if err != nil {
			t.Fatal(err)
		}
		expr, err := filter.AsSQL(f, outputsTable, tc.values)
		if err != nil {
			t.Fatal(err)
		}
		query, values := constructOutputsQuery(expr, tc.values, nowMillis, tc.after, 10)
		if query != tc.wantQuery {
			t.Errorf("case %d: got %s want %s", i, query, tc.wantQuery)
		}
		if !testutil.DeepEqual(values, tc.wantValues) {
			t.Errorf("case %d: got %#v, want %#v", i, values, tc.wantValues)
		}
	}
}
