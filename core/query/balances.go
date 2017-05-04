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
func (ind *Indexer) Balances(ctx context.Context, filt string, vals []interface{}, sumBy []filter.Field, timestampMS uint64) ([]interface{}, error) {
	p, err := filter.Parse(filt, outputsTable, vals)
	if err != nil {
		return nil, err
	}
	if len(vals) != p.Parameters {
		return nil, ErrParameterCountMismatch
	}
	expr, err := filter.AsSQL(p, outputsTable, vals)
	if err != nil {
		return nil, err
	}
	queryStr, queryArgs, err := constructBalancesQuery(expr, vals, sumBy, timestampMS)
	if err != nil {
		return nil, err
	}
	rows, err := ind.db.QueryContext(ctx, queryStr, queryArgs...)
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
		// This struct enforces JSON field ordering in API output.
		item := struct {
			SumBy  map[string]interface{} `json:"sum_by,omitempty"`
			Amount uint64                 `json:"amount"`
		}{
			Amount: balance,
		}
		if len(sumByValues) > 0 {
			item.SumBy = sumByValues
		}
		balances = append(balances, item)
	}
	return balances, errors.Wrap(rows.Err())
}

func constructBalancesQuery(expr string, vals []interface{}, sumBy []filter.Field, timestampMS uint64) (string, []interface{}, error) {
	var buf bytes.Buffer

	buf.WriteString("SELECT COALESCE(SUM(amount), 0)")
	for _, field := range sumBy {
		fieldSQL, err := filter.FieldAsSQL(outputsTable, field)
		if err != nil {
			return "", nil, err
		}

		buf.WriteString(", ")
		buf.WriteString(fieldSQL)
	}
	buf.WriteString(" FROM ")
	buf.WriteString(pq.QuoteIdentifier("annotated_outputs"))
	buf.WriteString(" AS out WHERE ")
	if len(expr) > 0 {
		buf.WriteString("(")
		buf.WriteString(expr)
		buf.WriteString(") AND ")
	}

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
	return buf.String(), vals, nil
}
