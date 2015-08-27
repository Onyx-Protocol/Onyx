package wallet

import "database/sql"

const (
	chainPaymentNamespace  = 0
	chainPasscodeNamespace = 1
	chainAssetsNamespace   = 2

	customerPaymentNamespace = 0
	customerAssetsNamespace  = 1
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
