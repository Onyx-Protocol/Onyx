package query

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"chain/core/cursor"
	"chain/errors"
)

// Cursors queries the blockchain for cursors matching the query.
func (ind *Indexer) Cursors(ctx context.Context, after string, limit int) ([]*cursor.Cursor, string, error) {
	queryStr, queryArgs := constructCursorsQuery(after, limit)
	rows, err := ind.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, "", errors.Wrap(err, "executing cursors query")
	}
	defer rows.Close()

	cursors := make([]*cursor.Cursor, 0, limit)
	for rows.Next() {
		var (
			cur         cursor.Cursor
			alias       sql.NullString
			isAscending bool
		)
		err := rows.Scan(&cur.ID, &alias, &cur.Filter, &cur.After, &isAscending)
		if err != nil {
			return nil, "", errors.Wrap(err, "scanning cursor row")
		}

		if isAscending {
			cur.Order = "asc"
		} else {
			cur.Order = "desc"
		}

		if alias.Valid {
			cur.Alias = &alias.String
		}

		after = cur.ID
		cursors = append(cursors, &cur)
	}
	err = rows.Err()
	if err != nil {
		return nil, "", errors.Wrap(err)
	}

	return cursors, after, nil
}

func constructCursorsQuery(after string, limit int) (string, []interface{}) {
	var vals []interface{}

	q := "SELECT id, alias, filter, after, is_ascending FROM cursors WHERE "
	// add after conditions
	q += fmt.Sprintf("($%d='' OR id < $%d) ", len(vals)+1, len(vals)+1)
	vals = append(vals, after)

	q += "ORDER BY id DESC "
	q += "LIMIT " + strconv.Itoa(limit)
	return q, vals
}
