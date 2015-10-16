// Package appdb provides low-level database operations
// for Chain enterprise objects.
//
// All functions in this package can return errors "wrapped"
// with more information using chain/errors.
package appdb

import (
	"database/sql"
	"errors"
)

// ErrBadLabel is returned by functions that operate
// on the label of various objects, for instance
// nodes and accounts.
var ErrBadLabel = errors.New("bad label")

// Init creates some objects in db.
// It must be called on program start,
// before any other functions in this package.
func Init(db *sql.DB) error {
	_, err := db.Exec(keySQL)
	return err
}
