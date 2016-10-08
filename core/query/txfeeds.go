package query

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"chain/core/txfeed"
	"chain/errors"
)

// TxFeeds queries the blockchain for txfeeds matching the query.
func (ind *Indexer) TxFeeds(ctx context.Context, after string, limit int) ([]*txfeed.TxFeed, string, error) {
	queryStr, queryArgs := constructTxFeedsQuery(after, limit)
	rows, err := ind.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, "", errors.Wrap(err, "executing txfeeds query")
	}
	defer rows.Close()

	txfeeds := make([]*txfeed.TxFeed, 0, limit)
	for rows.Next() {
		var (
			feed  txfeed.TxFeed
			alias sql.NullString
		)
		err := rows.Scan(&feed.ID, &alias, &feed.Filter, &feed.After)
		if err != nil {
			return nil, "", errors.Wrap(err, "scanning txfeed row")
		}

		if alias.Valid {
			feed.Alias = &alias.String
		}

		after = feed.ID
		txfeeds = append(txfeeds, &feed)
	}
	err = rows.Err()
	if err != nil {
		return nil, "", errors.Wrap(err)
	}

	return txfeeds, after, nil
}

func constructTxFeedsQuery(after string, limit int) (string, []interface{}) {
	var vals []interface{}

	q := "SELECT id, alias, filter, after FROM txfeeds WHERE "
	// add after conditions
	q += fmt.Sprintf("($%d='' OR id < $%d) ", len(vals)+1, len(vals)+1)
	vals = append(vals, after)

	q += "ORDER BY id DESC "
	q += "LIMIT " + strconv.Itoa(limit)
	return q, vals
}
