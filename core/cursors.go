package core

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"chain/core/query"
	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/httpjson"
)

type Cursor struct {
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias,omitempty"`
	Filter string `json:"filter,omitempty"`
	After  string `json:"after,omitempty"`
	Order  string `json:"order,omitempty"`
}

// POST /create-cursor
func (a *api) createCursor(ctx context.Context, in struct {
	Alias  string
	Filter string

	// ClientToken is the application's unique token for the cursor. Every cursor
	// should have a unique client token. The client token is used to ensure
	// idempotency of create cursor requests. Duplicate create cursor requests
	// with the same client_token will only create one cursor.
	ClientToken *string `json:"client_token"`
}) (*Cursor, error) {
	defer metrics.RecordElapsed(time.Now())

	after := fmt.Sprintf("%x:%x-%x", a.c.Height(), math.MaxInt32, uint64(math.MaxInt64))
	cur := &Cursor{
		Alias:  in.Alias,
		Filter: in.Filter,
		After:  after,
	}

	return insertCursor(ctx, cur, in.ClientToken)
}

// insertCursor adds the cursor to the database. If the cursor has a client token,
// and there already exists a cursor with that client token, insertCursor will
// lookup and return the existing cursor instead.
func insertCursor(ctx context.Context, cur *Cursor, clientToken *string) (*Cursor, error) {
	const q = `
		INSERT INTO cursors (alias, filter, after, is_ascending, client_token)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (client_token) DO NOTHING
		RETURNING id
	`

	isAscending := cur.Order == "asc"

	err := pg.QueryRow(
		ctx, q, cur.Alias, cur.Filter, cur.After,
		isAscending, clientToken).Scan(&cur.ID)

	if pg.IsUniqueViolation(err) {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "non-unique alias")
	} else if err == sql.ErrNoRows && clientToken != nil {
		// There is already a cursor with the provided client
		// token. We should return the existing cursor
		cur, err = cursorByClientToken(ctx, *clientToken)
		if err != nil {
			return nil, errors.Wrap(err, "retrieving existing cursor")
		}
	} else if err != nil {
		return nil, err
	}

	return cur, nil
}

func cursorByClientToken(ctx context.Context, clientToken string) (*Cursor, error) {
	const q = `
		SELECT id, alias, filter, after, is_ascending
		FROM cursors
		WHERE client_token=$1
	`

	var (
		cur         Cursor
		isAscending bool
	)
	err := pg.QueryRow(ctx, q, clientToken).Scan(&cur.ID, &cur.Alias, &cur.Filter, &cur.After, &isAscending)
	if err != nil {
		return nil, err
	}

	if isAscending {
		cur.Order = "asc"
	} else {
		cur.Order = "desc"
	}

	return &cur, nil
}

// POST /get-cursor
func getCursor(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}) (*Cursor, error) {
	defer metrics.RecordElapsed(time.Now())

	where := ` WHERE `
	id := in.ID
	if in.ID != "" {
		where += `id=$1`
	} else {
		where += `alias=$1`
		id = in.Alias
	}

	q := `
		SELECT id, alias, filter, after, is_ascending
		FROM cursors
	` + where

	var (
		cur         Cursor
		isAscending bool
	)

	err := pg.QueryRow(ctx, q, id).Scan(&cur.ID, &cur.Alias, &cur.Filter, &cur.After, &isAscending)
	if err != nil {
		return nil, err
	}

	if isAscending {
		cur.Order = "asc"
	} else {
		cur.Order = "desc"
	}

	return &cur, nil
}

// POST /update-cursor
func updateCursor(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
	Prev  string `json:"prev"`
	After string `json:"after"`
}) (*Cursor, error) {
	defer metrics.RecordElapsed(time.Now())

	bad, err := isBefore(in.After, in.Prev)
	if err != nil {
		return nil, err
	}

	if bad {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "new After cannot be before Prev")
	}

	where := ` WHERE `
	id := in.ID
	if in.ID != "" {
		where += `id=$2`
	} else {
		where += `alias=$2`
		id = in.Alias
	}

	q := `
		UPDATE cursors SET after=$1
	` + where + ` AND after=$3`

	res, err := pg.Exec(ctx, q, in.After, id, in.Prev)
	if err != nil {
		return nil, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	if affected == 0 {
		return nil, errors.WithDetailf(errNotFound, "could not find cursor with id/alias=%s and prev=%s", id, in.Prev)
	}

	return &Cursor{
		ID:    in.ID,
		Alias: in.Alias,
		After: in.After,
	}, nil
}

// isBefore returns true if a is before b. It returns an error if either
// a or b are not valid query.TxAfters.
func isBefore(a, b string) (bool, error) {
	aAfter, err := query.DecodeTxAfter(a)
	if err != nil {
		return false, err
	}

	bAfter, err := query.DecodeTxAfter(b)
	if err != nil {
		return false, err
	}

	return aAfter.FromBlockHeight < bAfter.FromBlockHeight ||
		(aAfter.FromBlockHeight == bAfter.FromBlockHeight &&
			aAfter.FromPosition < bAfter.FromPosition), nil
}
