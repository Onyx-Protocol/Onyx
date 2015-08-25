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
