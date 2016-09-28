package accesstoken

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
	"time"

	"golang.org/x/crypto/sha3"

	"chain/database/pg"
	"chain/errors"
)

const tokenSize = 32

var (
	// ErrBadID is returned when Create is called on an invalid id string.
	ErrBadID = errors.New("invalid id")
	// ErrDuplicateID is returned when Create is called on an existing ID.
	ErrDuplicateID = errors.New("duplicate access token ID")
	// ErrBadType is returned when Create is called with a bad type.
	ErrBadType = errors.New("type must be client or network")

	defaultLimit = 100

	// validIDRegexp checks that all characters are alphumeric, _ or -.
	// It also must have a length of at least 1.
	validIDRegexp = regexp.MustCompile(`^[\w-]+$`)
)

type Token struct {
	ID      string    `json:"id"`
	Token   string    `json:"token,omitempty"`
	Type    string    `json:"type"`
	Created time.Time `json:"created_at"`
	sortID  string
}

// Create generates a new access token with the given ID.
func Create(ctx context.Context, id, typ string) (*Token, error) {
	if !validIDRegexp.MatchString(id) {
		return nil, errors.WithDetailf(ErrBadID, "invalid id %q", id)
	}

	if typ != "client" && typ != "network" {
		return nil, errors.WithDetailf(ErrBadType, "unknown type %q", typ)
	}

	var secret [tokenSize]byte
	_, err := rand.Read(secret[:])
	if err != nil {
		return nil, err
	}
	hashedSecret := sha3.Sum256(secret[:])

	const q = `
		INSERT INTO access_tokens (id, type, hashed_secret)
		VALUES($1, $2, $3)
		RETURNING created, sort_id
	`
	var (
		created time.Time
		sortID  string
	)
	err = pg.QueryRow(ctx, q, id, typ, hashedSecret[:]).Scan(&created, &sortID)
	if pg.IsUniqueViolation(err) {
		return nil, errors.WithDetailf(ErrDuplicateID, "id %q already in use", id)
	}
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &Token{
		ID:      id,
		Token:   fmt.Sprintf("%s:%x", id, secret),
		Type:    typ,
		Created: created,
		sortID:  sortID,
	}, nil
}

// Check returns whether or not an id-secret pair is a valid access token.
func Check(ctx context.Context, id, typ string, secret []byte) (bool, error) {
	var toHash [tokenSize]byte
	copy(toHash[:], secret)

	hashed := sha3.Sum256(toHash[:])

	const q = `SELECT EXISTS(SELECT 1 FROM access_tokens WHERE id=$1 AND type=$2 AND hashed_secret=$3)`
	var valid bool
	err := pg.QueryRow(ctx, q, id, typ, hashed[:]).Scan(&valid)
	if err != nil {
		return false, err
	}

	return valid, nil
}

// List lists all access tokens.
func List(ctx context.Context, after string, limit int) ([]*Token, string, error) {
	if limit == 0 {
		limit = defaultLimit
	}
	const q = `
		SELECT id, type, sort_id, created FROM access_tokens
		WHERE ($1='' OR sort_id<$1)
		ORDER BY sort_id DESC
		LIMIT $2
	`
	var tokens []*Token
	err := pg.ForQueryRows(ctx, q, after, limit, func(id, typ, sortID string, created time.Time) {
		tokens = append(tokens, &Token{
			ID:      id,
			Type:    typ,
			Created: created,
			sortID:  sortID,
		})
	})
	if err != nil {
		return nil, "", errors.Wrap(err)
	}

	var next string
	if len(tokens) > 0 {
		next = tokens[len(tokens)-1].sortID
	}

	return tokens, next, nil
}

// ClientTokenExists returns whether or not a client token is present
// in the database.
func ClientTokenExists(ctx context.Context) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM access_tokens WHERE type='client')`
	var exists bool
	err := pg.QueryRow(ctx, q).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err)
	}
	return exists, nil
}

// Delete deletes an access token by id.
func Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM access_tokens WHERE id=$1`
	res, err := pg.Exec(ctx, q, id)
	if err != nil {
		return errors.Wrap(err)
	}

	deleted, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err)
	}

	if deleted == 0 {
		return errors.WithDetailf(pg.ErrUserInputNotFound, "acccess token id %s", id)
	}
	return nil
}
