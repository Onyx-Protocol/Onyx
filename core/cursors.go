package core

import (
	"context"
	"database/sql"
	"time"

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
func createCursor(ctx context.Context, in struct {
	Alias  string
	Filter string

	// ClientToken is the application's unique token for the cursor. Every cursor
	// should have a unique client token. The client token is used to ensure
	// idempotency of create cursor requests. Duplicate create cursor requests
	// with the same client_token will only create one cursor.
	ClientToken *string `json:"client_token"`
}) (*Cursor, error) {
	defer metrics.RecordElapsed(time.Now())

	cur := &Cursor{
		Alias:  in.Alias,
		Filter: in.Filter,
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
