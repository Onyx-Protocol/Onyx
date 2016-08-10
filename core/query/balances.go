package query

import (
	"bytes"
	"strconv"

	"github.com/lib/pq"

	"chain/core/query/chql"
)

func constructBalancesQuery(expr chql.SQLExpr) (string, []interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("SELECT SUM((data->>'amount')::integer) AS balance")
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
	buf.WriteString(expr.SQL)
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
	return buf.String(), expr.Values, nil
}
