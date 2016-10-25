// Package txfeed implements Chain Core's transaction feeds.
package txfeed

import (
	"context"
	"database/sql"

	"chain/core/query/filter"
	"chain/database/pg"
	"chain/errors"
	"chain/net/http/httpjson"
)

type TxFeed struct {
	ID     string  `json:"id,omitempty"`
	Alias  *string `json:"alias"`
	Filter string  `json:"filter,omitempty"`
	After  string  `json:"after,omitempty"`
}

func Create(ctx context.Context, alias, fil, after string, clientToken *string) (*TxFeed, error) {
	// Validate the filter.
	_, err := filter.Parse(fil)
	if err != nil {
		return nil, err
	}

	var ptrAlias *string
	if alias != "" {
		ptrAlias = &alias
	}

	feed := &TxFeed{
		Alias:  ptrAlias,
		Filter: fil,
		After:  after,
	}

	db := pg.FromContext(ctx)
	return insertTxFeed(ctx, db, feed, clientToken)
}

// insertTxFeed adds the txfeed to the database. If the txfeed has a client token,
// and there already exists a txfeed with that client token, insertTxFeed will
// lookup and return the existing txfeed instead.
func insertTxFeed(ctx context.Context, db pg.DB, feed *TxFeed, clientToken *string) (*TxFeed, error) {
	const q = `
		INSERT INTO txfeeds (alias, filter, after, client_token)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (client_token) DO NOTHING
		RETURNING id
	`

	var alias sql.NullString
	if feed.Alias != nil {
		alias = sql.NullString{Valid: true, String: *feed.Alias}
	}

	err := db.QueryRow(
		ctx, q, alias, feed.Filter, feed.After,
		clientToken).Scan(&feed.ID)

	if pg.IsUniqueViolation(err) {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "non-unique alias")
	} else if err == sql.ErrNoRows && clientToken != nil {
		// There is already a txfeed with the provided client
		// token. We should return the existing txfeed
		feed, err = txfeedByClientToken(ctx, db, *clientToken)
		if err != nil {
			return nil, errors.Wrap(err, "retrieving existing txfeed")
		}
	} else if err != nil {
		return nil, err
	}

	return feed, nil
}

func txfeedByClientToken(ctx context.Context, db pg.DB, clientToken string) (*TxFeed, error) {
	const q = `
		SELECT id, alias, filter, after
		FROM txfeeds
		WHERE client_token=$1
	`

	var (
		feed  TxFeed
		alias sql.NullString
	)
	err := db.QueryRow(ctx, q, clientToken).Scan(&feed.ID, &alias, &feed.Filter, &feed.After)
	if err != nil {
		return nil, err
	}

	if alias.Valid {
		feed.Alias = &alias.String
	}

	return &feed, nil
}

func Find(ctx context.Context, id, alias string) (*TxFeed, error) {
	where := ` WHERE `
	if id != "" {
		where += `id=$1`
	} else {
		where += `alias=$1`
		id = alias
	}

	q := `
		SELECT id, alias, filter, after
		FROM txfeeds
	` + where

	var (
		feed     TxFeed
		sqlAlias sql.NullString
	)

	err := pg.FromContext(ctx).QueryRow(ctx, q, id).Scan(&feed.ID, &sqlAlias, &feed.Filter, &feed.After)
	if err != nil {
		return nil, err
	}

	if sqlAlias.Valid {
		feed.Alias = &sqlAlias.String
	}

	return &feed, nil
}

func Delete(ctx context.Context, id, alias string) error {
	where := ` WHERE `
	if id != "" {
		where += `id=$1`
	} else {
		where += `alias=$1`
		id = alias
	}

	q := `DELETE FROM txfeeds` + where

	res, err := pg.FromContext(ctx).Exec(ctx, q, id)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return errors.WithDetailf(pg.ErrUserInputNotFound, "could not find and delete txfeed with id/alias=%s", id)
	}

	return nil
}

func Update(ctx context.Context, id, alias, after, prev string) (*TxFeed, error) {
	where := ` WHERE `
	if id != "" {
		where += `id=$2`
	} else {
		where += `alias=$2`
		id = alias
	}

	q := `
		UPDATE txfeeds SET after=$1
	` + where + ` AND after=$3`

	res, err := pg.FromContext(ctx).Exec(ctx, q, after, id, prev)
	if err != nil {
		return nil, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	if affected == 0 {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "could not find txfeed with id/alias=%s and prev=%s", id, prev)
	}

	return &TxFeed{
		ID:    id,
		Alias: &alias,
		After: after,
	}, nil
}
