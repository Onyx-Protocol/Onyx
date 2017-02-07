package filter

import (
	"testing"

	"chain/errors"
	"chain/testutil"
)

var (
	inputsSQLTable = &SQLTable{
		Name:  "annotated_inputs",
		Alias: "inp",
		Columns: map[string]*SQLColumn{
			"a":            {Name: "a", Type: String, SQLType: SQLText},
			"b":            {Name: "b", Type: String, SQLType: SQLText},
			"type":         {Name: "type", Type: String, SQLType: SQLText},
			"amount":       {Name: "amount", Type: Integer, SQLType: SQLBigint},
			"asset_id":     {Name: "asset_id", Type: String, SQLType: SQLBytea},
			"account_tags": {Name: "account_tags", Type: Object, SQLType: SQLJSONB},
		},
		ForeignKeys: map[string]*SQLForeignKey{},
	}
	outputsSQLTable = &SQLTable{
		Name:  "annotated_outputs",
		Alias: "out",
		Columns: map[string]*SQLColumn{
			"b": {Name: "b", Type: String, SQLType: SQLText},
		},
		ForeignKeys: map[string]*SQLForeignKey{},
	}
	transactionsSQLTable = &SQLTable{
		Name:  "annotated_txs",
		Alias: "txs",
		Columns: map[string]*SQLColumn{
			"id":       {Name: "tx_hash", Type: String, SQLType: SQLBytea},
			"ref":      {Name: "ref", Type: Object, SQLType: SQLJSONB},
			"position": {Name: "position", Type: Integer, SQLType: SQLInteger},
			"is_local": {Name: "local", Type: Bool, SQLType: SQLBool},
		},
		ForeignKeys: map[string]*SQLForeignKey{
			"inputs":  {Table: inputsSQLTable, LocalColumn: "tx_hash", ForeignColumn: "tx_hash"},
			"outputs": {Table: outputsSQLTable, LocalColumn: "tx_hash", ForeignColumn: "tx_hash"},
		},
	}
)

func TestFieldAsSQL(t *testing.T) {
	testCases := []struct {
		tbl   *SQLTable
		field string
		sql   string
	}{
		{tbl: inputsSQLTable, field: `a`, sql: `inp."a"`},
		{tbl: inputsSQLTable, field: `asset_id`, sql: `encode(inp."asset_id", 'hex')`},
		{tbl: transactionsSQLTable, field: `ref.buyer.address.state`, sql: `txs."ref"->'buyer'->'address'->>'state'`},
	}

	for _, tc := range testCases {
		f, err := ParseField(tc.field)
		if err != nil {
			t.Fatal(err)
		}

		got, err := FieldAsSQL(tc.tbl, f)
		if err != nil {
			t.Error(err)
			continue
		}
		if got != tc.sql {
			t.Errorf("FieldAsSQL(%s) = %s, want %s", tc.field, got, tc.sql)
		}
	}
}

func TestAsSQL(t *testing.T) {
	testCases := []struct {
		q   string
		sql string
		tbl *SQLTable
		err error
	}{
		{ // empty predicate
		},
		{ // boolean attribute
			q:   `is_local`,
			tbl: transactionsSQLTable,
			sql: `txs."local"`,
		},
		{ // error - invalid attribute
			q:   `garbage`,
			tbl: transactionsSQLTable,
			err: errors.WithDetail(ErrBadFilter, "invalid attribute: garbage"),
		},
		{ // bytea columns
			q:   `asset_id = $1`,
			tbl: inputsSQLTable,
			sql: `encode(inp."asset_id", 'hex') = $1`,
		},
		{ // paren expressions
			q:   `(asset_id = $1)`,
			tbl: inputsSQLTable,
			sql: `(encode(inp."asset_id", 'hex') = $1)`,
		},
		{ // indexing into arbitrary json
			q:   `ref.buyer.address.state = 'CA' AND ref.buyer.address.city = 'San Francisco'`,
			tbl: transactionsSQLTable,
			sql: `(txs."ref"->'buyer'->'address'->>'state') = 'CA' AND (txs."ref"->'buyer'->'address'->>'city') = 'San Francisco'`,
		},
		{ // indexing into arbitrary json as an integer
			q:   `ref.buyer.address.street_number = 200`,
			tbl: transactionsSQLTable,
			sql: `(txs."ref"->'buyer'->'address'->>'street_number')::bigint = 200::bigint`,
		},
		{ // indexing into arbitrary json as a boolean
			q:   `ref.buyer.is_high_priority`,
			tbl: transactionsSQLTable,
			sql: `(txs."ref"->'buyer'->>'is_high_priority')::boolean`,
		},
		{ // error - indexing into non-json attribute
			q:   `is_local.but_really`,
			tbl: transactionsSQLTable,
			err: errors.WithDetail(ErrBadFilter, "cannot index on non-object attribute: is_local"),
		},
		{ // error - unbound parameter
			q:   `asset_id = $2`, // $2 too big; only 1 param given
			tbl: inputsSQLTable,
			err: errors.WithDetail(ErrBadFilter, "unbound placeholder: $2"),
		},
		{ // integer to biginteger conversion
			q:   `position = 2`,
			tbl: transactionsSQLTable,
			sql: `txs."position"::bigint = 2::bigint`,
		},
		{ // simple environment
			q:   `inputs(a = 'a' AND b = 'b')`,
			tbl: transactionsSQLTable,
			sql: `
EXISTS(SELECT 1 FROM annotated_inputs AS inp WHERE inp."tx_hash" = txs."tx_hash" AND (inp."a" = 'a' AND inp."b" = 'b'))
`,
		},
		{ // error - invalid environment
			q:   `inputs(asset_id = 'c001cafe')`,
			tbl: inputsSQLTable,
			err: errors.WithDetail(ErrBadFilter, "invalid environment `inputs`"),
		},
		{ // error - invalid attribute (in selectorExpr)
			q:   `data.asset_id = 'c001cafe'`,
			tbl: inputsSQLTable,
			err: errors.WithDetail(ErrBadFilter, "invalid attribute: data"),
		},
		{ // multiple environment expressions
			q:   `inputs(a = 'a') OR outputs(b = 'b')`,
			tbl: transactionsSQLTable,
			sql: `
EXISTS(SELECT 1 FROM annotated_inputs AS inp WHERE inp."tx_hash" = txs."tx_hash" AND (inp."a" = 'a'))
 OR 
EXISTS(SELECT 1 FROM annotated_outputs AS out WHERE out."tx_hash" = txs."tx_hash" AND (out."b" = 'b'))
`,
		},
		{ // environment expression and top-level expressions
			q:   `inputs(a = 'a') AND ref.txbankref = '1ab'`,
			tbl: transactionsSQLTable,
			sql: `
EXISTS(SELECT 1 FROM annotated_inputs AS inp WHERE inp."tx_hash" = txs."tx_hash" AND (inp."a" = 'a'))
 AND (txs."ref"->>'txbankref') = '1ab'`,
		},
	}

	values := []interface{}{"hey"}
	for _, tc := range testCases {
		p, err := Parse(tc.q, tc.tbl, values)
		if !testutil.DeepEqual(errors.Root(err), errors.Root(tc.err)) {
			t.Errorf("got error %q want error %q", err, tc.err)
		}
		if err != nil {
			continue
		}

		b := &sqlBuilder{baseTbl: tc.tbl, values: values, selectorTypes: p.selectorTypes}
		c := &sqlContext{sqlBuilder: b, tbl: tc.tbl}

		err = asSQL(c, p.expr)
		if !testutil.DeepEqual(errors.Root(err), errors.Root(tc.err)) {
			t.Errorf("got error %q want error %q", err, tc.err)
		}
		if err != nil && err.Error() != tc.err.Error() {
			t.Errorf("got error detail %q want error %q", errors.Detail(err), errors.Detail(tc.err))
		}
		if err == nil && c.buf.String() != tc.sql {
			t.Errorf("asSQL(%q) = %s, want %s", tc.q, c.buf.String(), tc.sql)
		}
	}
}
