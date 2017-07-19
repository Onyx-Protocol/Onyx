// Package accesstoken provides storage and validation of Chain Core
// credentials.
package accesstoken

// TODO(tessr): merge this package into chain/net/http/authn

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	"chain/crypto/sha3pool"
	"chain/database/pg"
	"chain/errors"
)

const (
	tokenSize    = 32
	defaultLimit = 100
)

var (
	// ErrBadID is returned when Create is called on an invalid id string.
	ErrBadID = errors.New("invalid id")
	// ErrDuplicateID is returned when Create is called on an existing ID.
	ErrDuplicateID = errors.New("duplicate access token ID")
	// ErrBadType is returned when Create is called with a bad type.
	ErrBadType = errors.New("type must be client or network")

	// validIDRegexp checks that all characters are alphumeric, _ or -.
	// It also must have a length of at least 1.
	validIDRegexp = regexp.MustCompile(`^[\w-]+$`)
)

type Token struct {
	ID      string    `json:"id"`
	Token   string    `json:"token,omitempty"`
	Type    string    `json:"type,omitempty"` // deprecated in 1.2
	Created time.Time `json:"created_at"`
	sortID  string
}

type CredentialStore struct {
	DB pg.DB
}

// Create generates a new access token with the given ID.
func (cs *CredentialStore) Create(ctx context.Context, id, typ string) (*Token, error) {
	if !validIDRegexp.MatchString(id) {
		return nil, errors.WithDetailf(ErrBadID, "invalid id %q", id)
	}

	var secret [tokenSize]byte
	_, err := rand.Read(secret[:])
	if err != nil {
		return nil, err
	}
	var hashedSecret [32]byte
	sha3pool.Sum256(hashedSecret[:], secret[:])

	const q = `
		INSERT INTO access_tokens (id, type, hashed_secret)
		VALUES($1, $2, $3)
		RETURNING created, sort_id
	`
	var (
		created   time.Time
		sortID    string
		maybeType = sql.NullString{String: typ, Valid: typ != ""}
	)
	err = cs.DB.QueryRowContext(ctx, q, id, maybeType, hashedSecret[:]).Scan(&created, &sortID)
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
func (cs *CredentialStore) Check(ctx context.Context, id string, secret []byte) (bool, error) {
	var (
		toHash [tokenSize]byte
		hashed [32]byte
	)
	copy(toHash[:], secret)
	sha3pool.Sum256(hashed[:], toHash[:])

	const q = `SELECT EXISTS(SELECT 1 FROM access_tokens WHERE id=$1 AND hashed_secret=$2)`
	var valid bool
	err := cs.DB.QueryRowContext(ctx, q, id, hashed[:]).Scan(&valid)
	if err != nil {
		return false, err
	}

	return valid, nil
}

// Exists returns whether an id is part of a valid access token. It does not validate a secret.
func (cs *CredentialStore) Exists(ctx context.Context, id string) bool {
	const q = `SELECT EXISTS(SELECT 1 FROM access_tokens WHERE id=$1)`
	var valid bool
	err := cs.DB.QueryRowContext(ctx, q, id).Scan(&valid)
	if err != nil {
		return false
	}
	return valid
}

// List lists all access tokens.
func (cs *CredentialStore) List(ctx context.Context, typ, after string, limit int) ([]*Token, string, error) {
	if limit == 0 {
		limit = defaultLimit
	}
	const q = `
		SELECT id, type, sort_id, created FROM access_tokens
		WHERE ($1='' OR type=$1::access_token_type) AND ($2='' OR sort_id<$2)
		ORDER BY sort_id DESC
		LIMIT $3
	`
	var tokens []*Token
	err := pg.ForQueryRows(ctx, cs.DB, q, typ, after, limit, func(id string, maybeType sql.NullString, sortID string, created time.Time) {
		t := Token{
			ID:      id,
			Created: created,
			Type:    maybeType.String,
			sortID:  sortID,
		}
		tokens = append(tokens, &t)
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

// Delete deletes an access token by id.
func (cs *CredentialStore) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM access_tokens WHERE id=$1`
	res, err := cs.DB.ExecContext(ctx, q, id)
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
