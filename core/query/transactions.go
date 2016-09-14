package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"chain/core/query/filter"
	"chain/errors"
)

var (
	ErrBadAfter               = errors.New("malformed pagination parameter after")
	ErrParameterCountMismatch = errors.New("wrong number of parameters to query")
)

type TxAfter struct {
	// FromBlockHeight and FromPosition uniquely identify the last transaction returned
	// by a list-transactions query.
	//
	// If list-transactions is called with a time range instead of a cursor, these fields
	// are populated with the position of the transaction at the start of the time range.
	FromBlockHeight uint64 // exclusive
	FromPosition    uint32 // exclusive

	// StopBlockHeight identifies the last block that should be included in a transaction
	// list. It is used when list-transactions is called with a time range instead
	// of a cursor.
	StopBlockHeight uint64 // inclusive
}

func (after TxAfter) String() string {
	return fmt.Sprintf("%x-%x-%x", after.FromBlockHeight, after.FromPosition, after.StopBlockHeight)
}

func DecodeTxAfter(str string) (c TxAfter, err error) {
	s := strings.Split(str, "-")
	if len(s) != 3 {
		return c, ErrBadAfter
	}
	from, err := strconv.ParseUint(s[0], 16, 64)
	if err != nil {
		return c, ErrBadAfter
	}
	pos, err := strconv.ParseUint(s[1], 16, 32)
	if err != nil {
		return c, ErrBadAfter
	}
	stop, err := strconv.ParseUint(s[2], 16, 64)
	if err != nil {
		return c, ErrBadAfter
	}
	return TxAfter{FromBlockHeight: from, FromPosition: uint32(pos), StopBlockHeight: stop}, nil
}

// LookupTxAfter looks up the transaction `after` for the provided time range.
func (ind *Indexer) LookupTxAfter(ctx context.Context, begin, end uint64) (TxAfter, error) {
	const q = `
		SELECT COALESCE(MAX(height), 0), COALESCE(MIN(height), 0) FROM query_blocks
		WHERE timestamp >= $1 AND timestamp <= $2
	`

	var from, stop uint64
	err := ind.db.QueryRow(ctx, q, begin, end).Scan(&from, &stop)
	if err != nil {
		return TxAfter{}, errors.Wrap(err, "querying `query_blocks`")
	}
	return TxAfter{
		FromBlockHeight: from,
		FromPosition:    math.MaxInt32, // TODO(tessr): Support reversing direction.
		StopBlockHeight: stop,
	}, nil
}

// Transactions queries the blockchain for transactions matching the
// filter predicate `p`.
func (ind *Indexer) Transactions(ctx context.Context, p filter.Predicate, vals []interface{}, cur TxAfter, limit int) ([]interface{}, *TxAfter, error) {
	if len(vals) != p.Parameters {
		return nil, nil, ErrParameterCountMismatch
	}
	expr, err := filter.AsSQL(p, "data", vals)
	if err != nil {
		return nil, nil, errors.Wrap(err, "converting to SQL")
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
		err := rows.Scan(&cur.FromBlockHeight, &cur.FromPosition, &data)
		if err != nil {
			return nil, nil, errors.Wrap(err, "scanning transaction row")
		}
		txns = append(txns, (*json.RawMessage)(&data))
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}
	return txns, &cur, nil
}

func constructTransactionsQuery(expr filter.SQLExpr, cur TxAfter, limit int) (string, []interface{}) {
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

	// add time range & after conditions
	buf.WriteString(fmt.Sprintf("(block_height, tx_pos) < ($%d, $%d) AND ", len(vals)+1, len(vals)+2))
	buf.WriteString(fmt.Sprintf("block_height >= $%d ", len(vals)+3))
	vals = append(vals, cur.FromBlockHeight, cur.FromPosition, cur.StopBlockHeight)

	buf.WriteString("ORDER BY block_height DESC, tx_pos DESC ")
	buf.WriteString("LIMIT " + strconv.Itoa(limit))
	return buf.String(), vals
}
