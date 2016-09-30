package txconsumer

import (
	"context"
	"database/sql"

	"chain/database/pg"
	"chain/errors"
	"chain/net/http/httpjson"
)

type TxConsumer struct {
	ID     string  `json:"id,omitempty"`
	Alias  *string `json:"alias"`
	Filter string  `json:"filter,omitempty"`
	After  string  `json:"after,omitempty"`
}

func Create(ctx context.Context, alias, filter, after string, clientToken *string) (*TxConsumer, error) {
	var ptrAlias *string
	if alias != "" {
		ptrAlias = &alias
	}

	consumer := &TxConsumer{
		Alias:  ptrAlias,
		Filter: filter,
		After:  after,
	}

	return insertTxConsumer(ctx, consumer, clientToken)
}

// insertTxConsumer adds the txconsumer to the database. If the txconsumer has a client token,
// and there already exists a txconsumer with that client token, insertTxConsumer will
// lookup and return the existing txconsumer instead.
func insertTxConsumer(ctx context.Context, consumer *TxConsumer, clientToken *string) (*TxConsumer, error) {
	const q = `
		INSERT INTO txconsumers (alias, filter, after, client_token)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (client_token) DO NOTHING
		RETURNING id
	`

	var alias sql.NullString
	if consumer.Alias != nil {
		alias = sql.NullString{Valid: true, String: *consumer.Alias}
	}

	err := pg.QueryRow(
		ctx, q, alias, consumer.Filter, consumer.After,
		clientToken).Scan(&consumer.ID)

	if pg.IsUniqueViolation(err) {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "non-unique alias")
	} else if err == sql.ErrNoRows && clientToken != nil {
		// There is already a txconsumer with the provided client
		// token. We should return the existing txconsumer
		consumer, err = txconsumerByClientToken(ctx, *clientToken)
		if err != nil {
			return nil, errors.Wrap(err, "retrieving existing txconsumer")
		}
	} else if err != nil {
		return nil, err
	}

	return consumer, nil
}

func txconsumerByClientToken(ctx context.Context, clientToken string) (*TxConsumer, error) {
	const q = `
		SELECT id, alias, filter, after
		FROM txconsumers
		WHERE client_token=$1
	`

	var (
		consumer TxConsumer
		alias    sql.NullString
	)
	err := pg.QueryRow(ctx, q, clientToken).Scan(&consumer.ID, &alias, &consumer.Filter, &consumer.After)
	if err != nil {
		return nil, err
	}

	if alias.Valid {
		consumer.Alias = &alias.String
	}

	return &consumer, nil
}

func Find(ctx context.Context, id, alias string) (*TxConsumer, error) {
	where := ` WHERE `
	if id != "" {
		where += `id=$1`
	} else {
		where += `alias=$1`
		id = alias
	}

	q := `
		SELECT id, alias, filter, after
		FROM txconsumers
	` + where

	var (
		consumer TxConsumer
		sqlAlias sql.NullString
	)

	err := pg.QueryRow(ctx, q, id).Scan(&consumer.ID, &sqlAlias, &consumer.Filter, &consumer.After)
	if err != nil {
		return nil, err
	}

	if sqlAlias.Valid {
		consumer.Alias = &sqlAlias.String
	}

	return &consumer, nil
}

func Delete(ctx context.Context, id, alias string) error {
	where := ` WHERE `
	if id != "" {
		where += `id=$1`
	} else {
		where += `alias=$1`
		id = alias
	}

	q := `DELETE FROM txconsumers` + where

	res, err := pg.Exec(ctx, q, id)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return errors.WithDetailf(pg.ErrUserInputNotFound, "could not find and delete txconsumer with id/alias=%s", id)
	}

	return nil
}

func Update(ctx context.Context, id, alias, after, prev string) (*TxConsumer, error) {
	where := ` WHERE `
	if id != "" {
		where += `id=$2`
	} else {
		where += `alias=$2`
		id = alias
	}

	q := `
		UPDATE txconsumers SET after=$1
	` + where + ` AND after=$3`

	res, err := pg.Exec(ctx, q, after, id, prev)
	if err != nil {
		return nil, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	if affected == 0 {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "could not find txconsumer with id/alias=%s and prev=%s", id, prev)
	}

	return &TxConsumer{
		ID:    id,
		Alias: &alias,
		After: after,
	}, nil
}
