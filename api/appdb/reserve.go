package appdb

import (
	"database/sql"
	"strings"

	"github.com/lib/pq"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/wire"
)

// ErrInsufficientFunds is an error for comparison purposes.
// It is returned when ReserveUTXOs cannot select the requested amount.
var ErrInsufficientFunds = errors.New("insufficient funds")

// UTXO is a simple wrapper around an output
// that contains its outpoint, amount and address id.
// It is used to create in input on a transaction.
type UTXO struct {
	OutPoint  *wire.OutPoint
	Amount    int64
	AddressID string
}

// ReserveUTXOs selects enough UTXOs to satisfy the requested amount.
// It returns ErrInsufficientFunds if the corresponding bucket
// does not have enough of the asset.
// ctx must have a database transaction.
func ReserveUTXOs(ctx context.Context, assetID, bucketID string, amount int64) ([]*UTXO, int64, error) {
	const q = `
		SELECT txid, index, amount, address_id
		FROM reserve_utxos($1, $2, $3)
	`

	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	rows, err := pg.FromContext(ctx).Query(q, assetID, bucketID, amount)
	return reserved(rows, err)
}

// ReserveTxUTXOs selects enough UTXOs to satisfy the requested amount.
// It returns ErrInsufficientFunds if the corresponding bucket
// and transaction do not have enough of the asset.
// ctx must have a database transaction.
func ReserveTxUTXOs(ctx context.Context, assetID, bucketID, txid string, amount int64) ([]*UTXO, int64, error) {
	const q = `
		SELECT txid, index, amount, address_id
		FROM reserve_tx_utxos($1, $2, $3, $4)
	`

	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	rows, err := pg.FromContext(ctx).Query(q, assetID, bucketID, txid, amount)
	return reserved(rows, err)
}

func reserved(rows *sql.Rows, err error) ([]*UTXO, int64, error) {
	if pqErr, ok := err.(*pq.Error); ok && strings.Contains(pqErr.Message, "insufficient funds") {
		return nil, 0, ErrInsufficientFunds
	} else if err != nil {
		return nil, 0, errors.Wrap(err, "reserving outputs")
	}
	defer rows.Close()

	var (
		utxos []*UTXO
		sum   int64
	)

	for rows.Next() {
		var (
			txid   string
			index  uint32
			uAmt   int64
			addrID string
		)
		err := rows.Scan(&txid, &index, &uAmt, &addrID)
		if err != nil {
			return nil, 0, errors.Wrap(err, "reserve outputs scan")
		}

		hash, err := wire.NewHash32FromStr(txid)
		if err != nil {
			return nil, 0, err
		}

		sum += uAmt
		utxos = append(utxos, &UTXO{wire.NewOutPoint(hash, index), uAmt, addrID})
	}
	if err = rows.Err(); err != nil {
		return nil, 0, errors.Wrap(err, "end row scan loop")
	}

	return utxos, sum, nil
}
