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

var (
	// ErrBadLabel is returned by functions that operate
	// on the label of various objects, for instance
	// nodes and accounts.
	ErrBadLabel = errors.New("bad label")

	// ErrCannotDelete means someone tried to delete an object (node,
	// account, asset, etc.) that is referenced by other objects.
	ErrCannotDelete = errors.New("cannot delete")
)

// Init creates some objects in db.
// It must be called on program start,
// before any other functions in this package.
func Init(db *sql.DB) error {
	_, err := db.Exec(keySQL)
	return err
}
