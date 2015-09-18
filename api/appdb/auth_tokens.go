package appdb

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
)

const (
	secretBytes           = 16
	tokenSecretBcryptCost = 10
)

// AuthToken represents an ID-secret pair. It can be used in the response of
// an API call, or to deserialize credentials for incoming API calls.
type AuthToken struct {
	ID        string    `json:"id"`
	Secret    string    `json:"secret,omitempty"`
	CreatedAt time.Time `json:"created_at"`
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
			RETURNING id, created_at
		`
		id        string
		createdAt time.Time
	)
	err = pg.FromContext(ctx).QueryRow(q, secretHash, typ, userID, expiresAt).Scan(&id, &createdAt)
	if err != nil {
		return nil, errors.Wrap(err, "insert query")
	}

	return &AuthToken{id, secret, createdAt}, nil
}

// GetAuthToken takes a token ID and returns the associated
// secret hash, user ID, and expiration.
func GetAuthToken(ctx context.Context, id string) (secretHash []byte, userID string, expiration time.Time, err error) {
	defer metrics.RecordElapsed(time.Now())
	var (
		q         = `SELECT secret_hash, user_id, expires_at FROM auth_tokens WHERE id = $1`
		expiresAt pq.NullTime
	)
	err = pg.FromContext(ctx).QueryRow(q, id).Scan(&secretHash, &userID, &expiresAt)
	if err != nil {
		return nil, "", expiresAt.Time, errors.Wrap(err, "select token")
	}
	return secretHash, userID, expiresAt.Time, nil
}

// ListAuthTokens returns a list of AuthTokens of the given type and owned by
// the given user.
func ListAuthTokens(ctx context.Context, userID string, typ string) ([]*AuthToken, error) {
	q := `
		SELECT id, created_at FROM auth_tokens
		WHERE user_id = $1 AND type = $2
		ORDER BY created_at
	`
	rows, err := pg.FromContext(ctx).Query(q, userID, typ)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var tokens []*AuthToken
	for rows.Next() {
		t := new(AuthToken)
		err := rows.Scan(&t.ID, &t.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		tokens = append(tokens, t)
	}

	if err := rows.Close(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return tokens, nil
}

// DeleteAuthToken removes the specified auth token from the database.
func DeleteAuthToken(ctx context.Context, id string) error {
	q := `DELETE FROM auth_tokens WHERE id = $1`
	_, err := pg.FromContext(ctx).Exec(q, id)
	if err != nil {
		return errors.Wrap(err, "delete query")
	}
	return nil
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
