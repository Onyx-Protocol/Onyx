// Package appdb provides low-level database operations
// for Chain app wallet objects.
package appdb

import "database/sql"

const (
	ChainPaymentNamespace  = 0
	ChainPasscodeNamespace = 1
	ChainAssetsNamespace   = 2

	CustomerPaymentNamespace = 0
	CustomerAssetsNamespace  = 1
)

// Init creates some objects in db.
// It must be called on program start,
// before any other functions in this package.
func Init(db *sql.DB) error {
	_, err := db.Exec(reserveSQL)
	if err != nil {
		return err
	}
	_, err = db.Exec(keySQL)
	return err
}
