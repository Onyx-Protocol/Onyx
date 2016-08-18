package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/lib/pq"

	"chain/core/query/chql"
)

type OutputsCursor struct {
	lastBlockHeight uint64
	lastTxPos       uint32
	lastIndex       uint32
}

func (cur OutputsCursor) String() string {
	return fmt.Sprintf("%x-%x-%x", cur.lastBlockHeight, cur.lastTxPos, cur.lastIndex)
}

func DecodeOutputsCursor(str string) (c *OutputsCursor, err error) {
	s := strings.Split(str, "-")
	if len(s) != 3 {
		return nil, ErrBadCursor
	}
	lastBlockHeight, err := strconv.ParseUint(s[0], 16, 64)
	if err != nil {
		return nil, ErrBadCursor
	}
	lastTxPos, err := strconv.ParseUint(s[1], 16, 32)
	if err != nil {
		return nil, ErrBadCursor
	}
	lastIndex, err := strconv.ParseUint(s[2], 16, 32)
	if err != nil {
		return nil, ErrBadCursor
	}
	return &OutputsCursor{
		lastBlockHeight: lastBlockHeight,
		lastTxPos:       uint32(lastTxPos),
		lastIndex:       uint32(lastIndex),
	}, nil
}

func (ind *Indexer) Outputs(ctx context.Context, q chql.Query, vals []interface{}, timestampMS uint64, cursor *OutputsCursor, limit int) ([]interface{}, *OutputsCursor, error) {
	expr, err := chql.AsSQL(q, "data", vals)
	if err != nil {
		return nil, nil, err
	}
	if len(expr.GroupBy) > 0 {
		return nil, nil, ErrMissingParameters
	}
	queryStr, queryArgs := constructOutputsQuery(expr, timestampMS, cursor, limit)
	rows, err := ind.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var newCursor OutputsCursor
	if cursor != nil {
		newCursor = *cursor
	}

	outputs := make([]interface{}, 0, limit)
	for rows.Next() {
		var (
			blockHeight uint64
			txPos       uint32
			index       uint32
			data        []byte
		)
		err = rows.Scan(&blockHeight, &txPos, &index, &data)
		if err != nil {
			return nil, nil, err
		}
		var output map[string]interface{}
		err = json.Unmarshal(data, &output)
		if err != nil {
			return nil, nil, err
		}
		outputs = append(outputs, output)

		newCursor.lastBlockHeight = blockHeight
		newCursor.lastTxPos = txPos
		newCursor.lastIndex = index
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, err
	}

	return outputs, &newCursor, nil
}

func constructOutputsQuery(expr chql.SQLExpr, timestampMS uint64, cursor *OutputsCursor, limit int) (string, []interface{}) {
	sql := fmt.Sprintf("SELECT block_height, tx_pos, output_index, data FROM %s", pq.QuoteIdentifier("annotated_outputs"))

	vals := make([]interface{}, 0, 4+len(expr.Values))
	vals = append(vals, expr.Values...)

	vals = append(vals, timestampMS)
	timestampValIndex := len(vals)

	where := strings.TrimSpace(expr.SQL)
	timespanExpr := fmt.Sprintf("timespan @> $%d::int8", timestampValIndex)
	if where == "" {
		where = timespanExpr
	} else {
		where = fmt.Sprintf("(%s) AND %s", where, timespanExpr)
	}

	if cursor != nil {
		vals = append(vals, cursor.lastBlockHeight)
		lastBlockHeightValIndex := len(vals)

		vals = append(vals, cursor.lastTxPos)
		lastTxPosValIndex := len(vals)

		vals = append(vals, cursor.lastIndex)
		lastIndexValIndex := len(vals)

		where = fmt.Sprintf("%s AND (block_height, tx_pos, output_index) > ($%d, $%d, $%d)", where, lastBlockHeightValIndex, lastTxPosValIndex, lastIndexValIndex)
	}

	sql += fmt.Sprintf(" WHERE %s ORDER BY block_height ASC, tx_pos ASC, output_index ASC LIMIT %d", where, limit)

	return sql, vals
}
