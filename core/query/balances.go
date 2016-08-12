package query

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/lib/pq"
	"golang.org/x/net/context"

	"chain/core/query/chql"
	"chain/errors"
)

// Balances performs a balances query against the annotated_outputs.
func (ind *Indexer) Balances(ctx context.Context, q chql.Query, vals []interface{}, timestampMS uint64) ([]interface{}, error) {
	expr, err := chql.AsSQL(q, "data", vals)
	if err != nil {
		return nil, err
	}
	queryStr, queryArgs := constructBalancesQuery(expr, timestampMS)
	rows, err := ind.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []interface{}
	for rows.Next() {
		// balance and groupings will hold the output of the row scan
		var balance uint64
		scanArguments := make([]interface{}, 0, len(expr.GroupBy)+1)
		scanArguments = append(scanArguments, &balance)
		for range expr.GroupBy {
			// TODO(jackson): Support grouping by things besides strings.
			scanArguments = append(scanArguments, new(string))
		}
		err := rows.Scan(scanArguments...)
		if err != nil {
			return nil, errors.Wrap(err, "scanning balance row")
		}

		item := map[string]interface{}{
			"amount": balance,
		}
		var groupBy []interface{}
		for i := range expr.GroupBy {
			groupBy = append(groupBy, scanArguments[i+1])
		}
		if len(groupBy) > 0 {
			item["group_by"] = groupBy
		}
		balances = append(balances, item)
	}
	return balances, errors.Wrap(rows.Err())
}

func constructBalancesQuery(expr chql.SQLExpr, timestampMS uint64) (string, []interface{}) {
	var buf bytes.Buffer

	buf.WriteString("SELECT COALESCE(SUM((data->>'amount')::integer), 0)")
	for i, grouping := range expr.GroupBy {
		buf.WriteString(", ")

		buf.WriteString(pq.QuoteIdentifier("data"))
		for _, field := range grouping {
			if i+1 < len(grouping) {
				buf.WriteString("->")
			} else {
				buf.WriteString("->>")
			}

			// Note, field here originally came from an identifier in ChQL, so
			// it should be safe to embed in a string without quoting.
			// TODO(jackson): Quote/restrict anyways to be defensive.
			buf.WriteString("'")
			buf.WriteString(field)
			buf.WriteString("'")
		}
	}
	buf.WriteString(" FROM ")
	buf.WriteString(pq.QuoteIdentifier("annotated_outputs"))
	buf.WriteString(" WHERE ")
	if len(expr.SQL) > 0 {
		buf.WriteString("(")
		buf.WriteString(expr.SQL)
		buf.WriteString(") AND ")
	}

	vals := make([]interface{}, 0, 1+len(expr.Values))
	vals = append(vals, expr.Values...)

	vals = append(vals, timestampMS)
	timestampValIndex := len(vals)

	buf.WriteString(fmt.Sprintf("timespan @> $%d::int8", timestampValIndex))

	if len(expr.GroupBy) > 0 {
		buf.WriteString(" GROUP BY ")
		for i := range expr.GroupBy {
			if i != 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(strconv.Itoa(i + 2)) // 1-indexed, skipping first col
		}
	}
	// TODO(jackson): Support pagination.
	return buf.String(), vals
}
