// Package txfeed implements Chain Core's transaction feeds.
package txfeed

import (
	"bytes"
	"context"
	"database/sql"

	"chain/core/query/filter"
	"chain/database/pg"
	"chain/errors"
)

var ErrDuplicateAlias = errors.New("duplicate feed alias")

type Tracker struct {
	DB pg.DB
}

type TxFeed struct {
	ID     string  `json:"id,omitempty"`
	Alias  *string `json:"alias"`
	Filter string  `json:"filter,omitempty"`
	After  string  `json:"after,omitempty"`
}

func (t *Tracker) Create(ctx context.Context, alias, fil, after string, clientToken *string) (*TxFeed, error) {
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
	return insertTxFeed(ctx, t.DB, feed, clientToken)
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
		return nil, errors.WithDetail(ErrDuplicateAlias, "a transaction feed with the provided alias already exists")
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

func (t *Tracker) Find(ctx context.Context, id, alias string) (*TxFeed, error) {
	var q bytes.Buffer

	q.WriteString(`
		SELECT id, alias, filter, after
		FROM txfeeds
		WHERE
	`)

	if id != "" {
		q.WriteString(`id=$1`)
	} else {
		q.WriteString(`alias=$1`)
		id = alias
	}

	var (
		feed     TxFeed
		sqlAlias sql.NullString
	)

	err := t.DB.QueryRow(ctx, q.String(), id).Scan(&feed.ID, &sqlAlias, &feed.Filter, &feed.After)
	if err != nil {
		return nil, err
	}

	if sqlAlias.Valid {
		feed.Alias = &sqlAlias.String
	}

	return &feed, nil
}

func (t *Tracker) Delete(ctx context.Context, id, alias string) error {
	var q bytes.Buffer

	q.WriteString(`DELETE FROM txfeeds WHERE `)

	if id != "" {
		q.WriteString(`id=$1`)
	} else {
		q.WriteString(`alias=$1`)
		id = alias
	}

	res, err := t.DB.Exec(ctx, q.String(), id)
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

func (t *Tracker) Update(ctx context.Context, id, alias, after, prev string) (*TxFeed, error) {
	var q bytes.Buffer

	q.WriteString(`UPDATE txfeeds SET after=$1 WHERE `)

	if id != "" {
		q.WriteString(`id=$2`)
	} else {
		q.WriteString(`alias=$2`)
		id = alias
	}

	q.WriteString(` AND after=$3`)

	res, err := t.DB.Exec(ctx, q.String(), after, id, prev)
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
