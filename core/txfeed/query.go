package txfeed

import (
	"context"
	"database/sql"
	"fmt"

	"chain/errors"
)

// Query queries the Chain Core for txfeeds matching the query.
func (t *Tracker) Query(ctx context.Context, after string, limit int) ([]*TxFeed, string, error) {
	const baseQ = `
		SELECT id, alias, filter, after FROM txfeeds
		WHERE ($1='' OR id < $1) ORDER BY id DESC LIMIT %d
	`
	rows, err := t.DB.QueryContext(ctx, fmt.Sprintf(baseQ, limit), after)
	if err != nil {
		return nil, "", errors.Wrap(err, "executing txfeeds query")
	}
	defer rows.Close()

	txfeeds := make([]*TxFeed, 0, limit)
	for rows.Next() {
		var (
			feed  TxFeed
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
