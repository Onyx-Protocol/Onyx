package query

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"golang.org/x/net/context"

	"chain/core/query/chql"
	"chain/errors"
)

var (
	ErrBadCursor         = errors.New("malformed pagination cursor")
	ErrMissingParameters = errors.New("missing parameters to query")
)

type TxCursor struct {
	MaxBlockHeight uint64 // inclusive
	MaxPosition    uint32 // inclusive
	MinBlockHeight uint64 // inclusive
}

func (cur TxCursor) String() string {
	return fmt.Sprintf("%x-%x-%x", cur.MaxBlockHeight, cur.MaxPosition, cur.MinBlockHeight)
}

func DecodeTxCursor(str string) (c TxCursor, err error) {
	s := strings.Split(str, "-")
	if len(s) != 3 {
		return c, ErrBadCursor
	}
	max, err := strconv.ParseUint(s[0], 16, 64)
	if err != nil {
		return c, ErrBadCursor
	}
	pos, err := strconv.ParseUint(s[1], 16, 32)
	if err != nil {
		return c, ErrBadCursor
	}
	min, err := strconv.ParseUint(s[2], 16, 64)
	if err != nil {
		return c, ErrBadCursor
	}
	return TxCursor{MaxBlockHeight: max, MaxPosition: uint32(pos), MinBlockHeight: min}, nil
}

// LookupTxCursor looks up the transaction cursor for the provided time range.
func (ind *Indexer) LookupTxCursor(ctx context.Context, begin, end uint64) (TxCursor, error) {
	const q = `
		SELECT COALESCE(MAX(height), 0), COALESCE(MIN(height), 0) FROM query_blocks
		WHERE timestamp >= $1 AND timestamp <= $2
	`

	var max, min uint64
	err := ind.db.QueryRow(ctx, q, begin, end).Scan(&max, &min)
	if err != nil {
		return TxCursor{}, errors.Wrap(err, "querying `query_blocks`")
	}
	return TxCursor{
		MaxBlockHeight: max,
		MaxPosition:    math.MaxInt32,
		MinBlockHeight: min,
	}, nil
}

// Transactions queries the blockchain for transactions matching the query `q`.
func (ind *Indexer) Transactions(ctx context.Context, q chql.Query, vals []interface{}, cur TxCursor, limit int) ([]interface{}, *TxCursor, error) {
	expr, err := chql.AsSQL(q, "data", vals)
	if err != nil {
		return nil, nil, errors.Wrap(err, "converting to SQL")
	}
	if len(expr.GroupBy) > 0 {
		// A GROUP BY query doesn't make sense for transactions. This
		// is caused by leaving a parameter unconstrained in the query.
		return nil, nil, ErrMissingParameters
	}

	queryStr, queryArgs := constructTransactionsQuery(expr, cur, limit)
	rows, err := ind.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "executing txn query")
	}
	defer rows.Close()

	txns := make([]interface{}, 0, limit)
	for rows.Next() {
		var data []byte
		err := rows.Scan(&cur.MaxBlockHeight, &cur.MaxPosition, &data)
		if err != nil {
			return nil, nil, errors.Wrap(err, "scanning transaction row")
		}

		// TODO(jackson): Use json.RawMessage?
		var m map[string]interface{}
		err = json.Unmarshal(data, &m)
		if err != nil {
			return nil, nil, err
		}
		txns = append(txns, m)
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}
	return txns, &cur, nil
}

func constructTransactionsQuery(expr chql.SQLExpr, cur TxCursor, limit int) (string, []interface{}) {
	var buf bytes.Buffer
	var vals []interface{}

	buf.WriteString("SELECT block_height, tx_pos, data FROM annotated_txs")
	buf.WriteString(" WHERE ")

	// add filter conditions
	if len(expr.SQL) > 0 {
		vals = append(vals, expr.Values...)
		buf.WriteString(expr.SQL)
		buf.WriteString(" AND ")
	}

	// add time range & cursor conditions
	buf.WriteString(fmt.Sprintf("(block_height, tx_pos) <= ($%d, $%d) AND ", len(vals)+1, len(vals)+2))
	buf.WriteString(fmt.Sprintf("block_height >= $%d ", len(vals)+3))
	vals = append(vals, cur.MaxBlockHeight, cur.MaxPosition, cur.MinBlockHeight)

	buf.WriteString("ORDER BY block_height DESC, tx_pos DESC ")
	buf.WriteString("LIMIT " + strconv.Itoa(limit))
	return buf.String(), vals
}
