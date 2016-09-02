package query

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/lib/pq"

	"chain/core/query/filter"
	"chain/errors"
)

// Balances performs a balances query against the annotated_outputs.
func (ind *Indexer) Balances(ctx context.Context, p filter.Predicate, vals []interface{}, sumBy []filter.Field, timestampMS uint64) ([]interface{}, error) {
	if len(vals) != p.Parameters {
		return nil, ErrParameterCountMismatch
	}
	expr, err := filter.AsSQL(p, "data", vals)
	if err != nil {
		return nil, err
	}
	queryStr, queryArgs := constructBalancesQuery(expr, sumBy, timestampMS)
	rows, err := ind.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []interface{}
	for rows.Next() {
		// balance and groupings will hold the output of the row scan
		var balance uint64
		scanArguments := make([]interface{}, 0, len(sumBy)+1)
		scanArguments = append(scanArguments, &balance)
		for range sumBy {
			// TODO(jackson): Support grouping by things besides strings.
			scanArguments = append(scanArguments, new(*string))
		}
		err := rows.Scan(scanArguments...)
		if err != nil {
			return nil, errors.Wrap(err, "scanning balance row")
		}

		sumByValues := map[string]interface{}{}
		for i, f := range sumBy {
			sumByValues[f.String()] = scanArguments[i+1]
		}
		item := map[string]interface{}{
			"amount": balance,
		}
		if len(sumByValues) > 0 {
			item["sum_by"] = sumByValues
		}
		balances = append(balances, item)
	}
	return balances, errors.Wrap(rows.Err())
}

func constructBalancesQuery(expr filter.SQLExpr, sumBy []filter.Field, timestampMS uint64) (string, []interface{}) {
	var buf bytes.Buffer

	buf.WriteString("SELECT COALESCE(SUM((data->>'amount')::bigint), 0)")
	for _, field := range sumBy {
		buf.WriteString(", ")
		buf.WriteString(filter.FieldAsSQL("data", field))
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

	if len(sumBy) > 0 {
		buf.WriteString(" GROUP BY ")
		for i := range sumBy {
			if i != 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(strconv.Itoa(i + 2)) // 1-indexed, skipping first col
		}
	}
	// TODO(jackson): Support pagination.
	return buf.String(), vals
}
