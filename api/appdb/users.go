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
	ErrBadEmail      = errors.New("bad email")
	ErrBadPassword   = errors.New("bad password")
	ErrPasswordCheck = errors.New("password does not match")
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
	if err := validateEmail(email); err != nil {
		return nil, err
	}

	if err := validatePassword(password); err != nil {
		return nil, err
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
	if pg.IsUniqueViolation(err) {
		return nil, errors.Wrap(ErrBadEmail, "email address already in use")
	}
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

// UpdateUserEmail modifies a user's email address. If the provided email
// address is not a valid email address, or the email address is in use by
// another user, ErrBadEmail is returned. As an extra layer of security, it will
// check the provided password; if the password is incorrect,
// ErrPasswordCheck is returned.
func UpdateUserEmail(ctx context.Context, id, password, email string) error {
	if err := validateEmail(email); err != nil {
		return err
	}

	if err := checkPassword(ctx, id, password); err != nil {
		return err
	}

	q := `UPDATE users SET email = $1 WHERE id = $2`
	_, err := pg.FromContext(ctx).Exec(q, email, id)
	if pg.IsUniqueViolation(err) {
		return errors.Wrap(ErrBadEmail, "email address already in use")
	}
	if err != nil {
		return errors.Wrap(err, "update query")
	}

	return nil
}

// UpdateUserPassword modifies a user's password. If the new password is not
// valid, ErrBadPassword is returned. As an extra layer of security, it will
// verify the current password; if the password is incorrect, ErrPasswordCheck
// is returned.
func UpdateUserPassword(ctx context.Context, id, password, newpass string) error {
	if err := validatePassword(newpass); err != nil {
		return err
	}

	if err := checkPassword(ctx, id, password); err != nil {
		return err
	}

	phash, err := bcrypt.GenerateFromPassword([]byte(newpass), passwordBcryptCost)
	if err != nil {
		return errors.Wrap(err, "password hash")
	}

	q := `UPDATE users SET password_hash = $1 WHERE id = $2`
	_, err = pg.FromContext(ctx).Exec(q, phash, id)
	if err != nil {
		return errors.Wrap(err, "update query")
	}

	return nil
}

func validateEmail(email string) error {
	switch {
	case len(email) > 255:
		return errors.WithDetail(ErrBadEmail, "too long")
	case !strings.Contains(email, "@"):
		return errors.WithDetail(ErrBadEmail, "no '@' symbol")
	}
	return nil
}

func validatePassword(password string) error {
	switch {
	case len(password) < 6:
		return errors.WithDetail(ErrBadPassword, "too short")
	case 255 < len(password):
		return errors.WithDetail(ErrBadPassword, "too long")
	}
	return nil
}

func checkPassword(ctx context.Context, id, password string) error {
	var (
		q     = `SELECT password_hash FROM users WHERE id = $1`
		phash []byte
	)
	err := pg.FromContext(ctx).QueryRow(q, id).Scan(&phash)
	if err != nil {
		return errors.Wrap(err, "select query")
	}

	if bcrypt.CompareHashAndPassword(phash, []byte(password)) != nil {
		return ErrPasswordCheck
	}

	return nil
}
