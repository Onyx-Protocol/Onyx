package wallets

import "database/sql"

const (
	chainPaymentNamespace  = 0
	chainPasscodeNamespace = 1
	chainAssetsNamespace   = 2

	customerPaymentNamespace = 0
	customerAssetsNamespace  = 1
)

var db *sql.DB

// Init initializes the package to talk to db.
// It must be called exactly once,
// before any other functions in this package.
func Init(sqldb *sql.DB) error {
	db = sqldb
	_, err := db.Exec(reserveSQL)
	if err != nil {
		return err
	}
	_, err = db.Exec(keySQL)
	return err
}
