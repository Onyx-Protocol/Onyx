package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

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
	// If list-transactions is called with a time range instead of an `after`, these fields
	// are populated with the position of the transaction at the start of the time range.
	FromBlockHeight uint64 // exclusive
	FromPosition    uint32 // exclusive

	// StopBlockHeight identifies the last block that should be included in a transaction
	// list. It is used when list-transactions is called with a time range instead
	// of an `after`.
	StopBlockHeight uint64 // inclusive
}

func (after TxAfter) String() string {
	return fmt.Sprintf("%d:%d-%d", after.FromBlockHeight, after.FromPosition, after.StopBlockHeight)
}

func DecodeTxAfter(str string) (c TxAfter, err error) {
	var from, pos, stop uint64
	_, err = fmt.Sscanf(str, "%d:%d-%d", &from, &pos, &stop)
	if err != nil {
		return c, errors.Wrap(ErrBadAfter, err.Error())
	}
	if from > math.MaxInt64 ||
		pos > math.MaxUint32 ||
		stop > math.MaxInt64 {
		return c, errors.Wrap(ErrBadAfter)
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
func (ind *Indexer) Transactions(ctx context.Context, p filter.Predicate, vals []interface{}, after TxAfter, limit int, asc bool) ([]interface{}, *TxAfter, error) {
	if len(vals) != p.Parameters {
		return nil, nil, ErrParameterCountMismatch
	}
	expr, err := filter.AsSQL(p, "data", vals)
	if err != nil {
		return nil, nil, errors.Wrap(err, "converting to SQL")
	}

	queryStr, queryArgs := constructTransactionsQuery(expr, after, asc, limit)

	if asc {
		return ind.waitForAndFetchTransactions(ctx, queryStr, queryArgs, after, limit)
	}
	return ind.fetchTransactions(ctx, queryStr, queryArgs, after, limit)
}

// If asc is true, the transactions will be returned from "in front" of the `after`
// param (e.g., the oldest transaction immediately after the `after` param,
// followed by the second oldest, etc) in ascending order.
func constructTransactionsQuery(expr filter.SQLExpr, after TxAfter, asc bool, limit int) (string, []interface{}) {
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

	if asc {
		// add time range & after conditions
		buf.WriteString(fmt.Sprintf("(block_height, tx_pos) > ($%d, $%d) AND ", len(vals)+1, len(vals)+2))
		buf.WriteString(fmt.Sprintf("block_height <= $%d ", len(vals)+3))
		vals = append(vals, after.FromBlockHeight, after.FromPosition, after.StopBlockHeight)

		buf.WriteString("ORDER BY block_height ASC, tx_pos ASC ")
	} else {
		// add time range & after conditions
		buf.WriteString(fmt.Sprintf("(block_height, tx_pos) < ($%d, $%d) AND ", len(vals)+1, len(vals)+2))
		buf.WriteString(fmt.Sprintf("block_height >= $%d ", len(vals)+3))
		vals = append(vals, after.FromBlockHeight, after.FromPosition, after.StopBlockHeight)

		buf.WriteString("ORDER BY block_height DESC, tx_pos DESC ")
	}

	buf.WriteString("LIMIT " + strconv.Itoa(limit))
	return buf.String(), vals
}

func (ind *Indexer) fetchTransactions(ctx context.Context, queryStr string, queryArgs []interface{}, after TxAfter, limit int) ([]interface{}, *TxAfter, error) {
	rows, err := ind.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "executing txn query")
	}
	defer rows.Close()

	txns := make([]interface{}, 0, limit)
	for rows.Next() {
		var data []byte
		err := rows.Scan(&after.FromBlockHeight, &after.FromPosition, &data)
		if err != nil {
			return nil, nil, errors.Wrap(err, "scanning transaction row")
		}
		txns = append(txns, (*json.RawMessage)(&data))
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}
	return txns, &after, nil
}

type fetchResp struct {
	txns  []interface{}
	after *TxAfter
	err   error
}

func (ind *Indexer) waitForAndFetchTransactions(ctx context.Context, queryStr string, queryArgs []interface{}, after TxAfter, limit int) ([]interface{}, *TxAfter, error) {
	resp := make(chan fetchResp, 1)
	go func() {
		var (
			txs []interface{}
			aft *TxAfter
			err error
		)

		for h := ind.c.Height(); len(txs) == 0; h++ {
			<-ind.pinStore.PinWaiter(TxPinName, h)
			if err != nil {
				resp <- fetchResp{nil, nil, err}
				return
			}

			txs, aft, err = ind.fetchTransactions(ctx, queryStr, queryArgs, after, limit)
			if err != nil {
				resp <- fetchResp{nil, nil, err}
				return
			}

			if len(txs) > 0 {
				resp <- fetchResp{txs, aft, nil}
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case r := <-resp:
		return r.txns, r.after, r.err
	}
}
