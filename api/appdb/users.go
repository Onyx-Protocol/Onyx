package appdb

import (
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
)

const passwordBcryptCost = 10

// User represents a single user. Instances should be safe to deliver in API
// responses.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// CreateUser creates a new row in the users table corresponding to the provided
// credentials.
func CreateUser(ctx context.Context, email, password string) (*User, error) {
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
