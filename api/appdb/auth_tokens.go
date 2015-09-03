package appdb

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/net/http/authn"
)

const (
	secretBytes           = 16
	tokenSecretBcryptCost = 10
)

// AuthToken represents an ID-secret pair. It can be used in the response of
// an API call, or to deserialize credentials for incoming API calls.
type AuthToken struct {
	ID     string `json:"id"`
	Secret string `json:"secret,omitempty"`
}

// CreateAuthToken creates an auth token for the given user. Conventional token
// types are "api" and "session". The token can be expiring or non-expiring; for
// non-expiring tokens, pass nil for the expiresAt parameter.
func CreateAuthToken(ctx context.Context, userID string, typ string, expiresAt *time.Time) (*AuthToken, error) {
	secret, secretHash, err := generateSecret()
	if err != nil {
		return nil, errors.Wrap(err, "generate token secret")
	}

	var (
		q = `
			INSERT INTO auth_tokens (secret_hash, type, user_id, expires_at)
			VALUES ($1, $2, $3, $4)
			RETURNING ID
		`
		id string
	)
	err = pg.FromContext(ctx).QueryRow(q, secretHash, typ, userID, expiresAt).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "insert query")
	}

	return &AuthToken{id, secret}, nil
}

// AuthenticateToken takes a token ID and secret and returns a user ID
// corresponding to those credentials. If the credentials are invalid,
// authn.ErrNotAuthenticated is returned.
func AuthenticateToken(ctx context.Context, id, secret string) (userID string, err error) {
	var (
		q          = `SELECT secret_hash, user_id, expires_at FROM auth_tokens WHERE id = $1`
		secretHash []byte
		uid        string
		expiresAt  pq.NullTime
	)
	err = pg.FromContext(ctx).QueryRow(q, id).Scan(&secretHash, &uid, &expiresAt)
	if err == sql.ErrNoRows {
		return "", authn.ErrNotAuthenticated
	}
	if err != nil {
		return "", errors.Wrap(err, "select token")
	}

	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		return "", authn.ErrNotAuthenticated
	}

	if bcrypt.CompareHashAndPassword(secretHash, []byte(secret)) != nil {
		return "", authn.ErrNotAuthenticated
	}

	return uid, nil
}

func generateSecret() (secret string, hash []byte, err error) {
	b := make([]byte, secretBytes)
	_, err = rand.Read(b)
	if err != nil {
		return "", nil, errors.Wrap(err, "generate random bytes")
	}

	secret = hex.EncodeToString(b)
	hash, err = bcrypt.GenerateFromPassword([]byte(secret), tokenSecretBcryptCost)
	if err != nil {
		return "", nil, errors.Wrap(err, "hash secret")
	}

	return secret, hash, nil
}
