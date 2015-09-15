package appdb

import (
	"database/sql"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/net/http/authn"
)

const passwordBcryptCost = 10

// Errors returned by CreateUser.
// May be wrapped using package chain/errors.
var (
	ErrBadEmail    = errors.New("bad email")
	ErrBadPassword = errors.New("bad password")
)

// User represents a single user. Instances should be safe to deliver in API
// responses.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// CreateUser creates a new row in the users table corresponding to the provided
// credentials.
func CreateUser(ctx context.Context, email, password string) (*User, error) {
	switch {
	case len(email) > 255:
		return nil, errors.WithDetail(ErrBadEmail, "too long")
	case !strings.Contains(email, "@"):
		return nil, errors.WithDetail(ErrBadEmail, "no '@' symbol")
	case len(password) < 6:
		return nil, errors.WithDetail(ErrBadPassword, "too short")
	case 255 < len(password):
		return nil, errors.WithDetail(ErrBadPassword, "too long")
	}

	phash, err := bcrypt.GenerateFromPassword([]byte(password), passwordBcryptCost)
	if err != nil {
		return nil, errors.Wrap(err, "password hash")
	}

	q := `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id
	`
	var id string
	err = pg.FromContext(ctx).QueryRow(q, email, phash).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "insert query")
	}

	return &User{id, email}, nil
}

// AuthenticateUserCreds takes an email and password and returns a user ID
// corresponding to those credentials. If the credentials are invalid,
// authn.ErrNotAuthenticated is returned.
func AuthenticateUserCreds(ctx context.Context, email, password string) (userID string, err error) {
	var (
		id    string
		phash []byte

		q = `SELECT id, password_hash FROM users WHERE lower(email) = lower($1)`
	)
	err = pg.FromContext(ctx).QueryRow(q, email).Scan(&id, &phash)
	if err == sql.ErrNoRows {
		return "", authn.ErrNotAuthenticated
	}
	if err != nil {
		return "", errors.Wrap(err, "select user")
	}

	if bcrypt.CompareHashAndPassword(phash, []byte(password)) != nil {
		return "", authn.ErrNotAuthenticated
	}

	return id, nil
}

// GetUser returns a User corresponding to the given ID. If no user is found,
// it will return an error with pg.ErrUserInputNotFound as its root.
func GetUser(ctx context.Context, id string) (*User, error) {
	var (
		q = `SELECT email FROM users WHERE id = $1`
		u = &User{ID: id}
	)

	err := pg.FromContext(ctx).QueryRow(q, id).Scan(&u.Email)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "ID: %v", id)
	}
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}

	return u, nil
}

// GetUserByEmail returns a User corresponding to the given email. It is not
// sensitive to the case of the provided email address. If no user is found,
// it will return an error with pg.ErrUserInputNotFound as its root.
func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var (
		q = `SELECT id, email FROM users WHERE lower(email) = lower($1)`
		u = new(User)
	)

	err := pg.FromContext(ctx).QueryRow(q, email).Scan(&u.ID, &u.Email)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "email: %v", email)
	}
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}

	return u, nil
}
