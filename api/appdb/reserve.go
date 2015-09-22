package appdb

import (
	"database/sql"
	"strings"
	"time"

	"github.com/lib/pq"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/wire"
	"chain/metrics"
)

// ErrInsufficientFunds is an error for comparison purposes.
// It is returned when ReserveUTXOs cannot select the requested amount.
var ErrInsufficientFunds = errors.New("insufficient funds")

// UTXO is a simple wrapper around an output
// that contains its outpoint, amount,
// and either address id or asset id.
// It is used to create in input on a transaction.
type UTXO struct {
	OutPoint  *wire.OutPoint
	Amount    int64
	AddressID string
	AssetID   string
}

// ReserveUTXOs selects enough UTXOs to satisfy the requested amount.
// It returns ErrInsufficientFunds if the corresponding bucket
// does not have enough of the asset.
// The reservation will be valid for the given ttl,
// after which the outputs will become available
// to reserve again.
// ctx must have a database transaction.
func ReserveUTXOs(ctx context.Context, assetID, bucketID string, amount int64, ttl time.Duration) ([]*UTXO, int64, error) {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		SELECT txid, index, amount, address_id
		FROM reserve_utxos($1, $2, $3, $4::interval)
	`

	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	rows, err := pg.FromContext(ctx).Query(q, assetID, bucketID, amount, ttl.String())
	return scanUTXOs(rows, err)
}

// ReserveTxUTXOs selects enough UTXOs to satisfy the requested amount.
// It returns ErrInsufficientFunds if the corresponding bucket
// and transaction do not have enough of the asset.
// The reservation will be valid for the given ttl,
// after which the outputs will become available
// to reserve again.
// ctx must have a database transaction.
func ReserveTxUTXOs(ctx context.Context, assetID, bucketID, txid string, amount int64, ttl time.Duration) ([]*UTXO, int64, error) {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		SELECT txid, index, amount, address_id
		FROM reserve_tx_utxos($1, $2, $3, $4, $5::interval)
	`

	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	rows, err := pg.FromContext(ctx).Query(q, assetID, bucketID, txid, amount, ttl.String())
	return scanUTXOs(rows, err)
}

func scanUTXOs(rows *sql.Rows, err error) ([]*UTXO, int64, error) {
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
		utxos = append(utxos, &UTXO{wire.NewOutPoint(hash, index), uAmt, addrID, ""})
	}
	if err = rows.Err(); err != nil {
		return nil, 0, errors.Wrap(err, "end row scan loop")
	}

	return utxos, sum, nil
}

// CancelReservations cancels reservations on all utxos
// specified by the outpoints. It does this by setting
// reserved_until to NOW(), effectively freeing the utxos.
func CancelReservations(ctx context.Context, outpoints []wire.OutPoint) error {
	var (
		hashes []string
		idxes  []uint32
	)
	for _, op := range outpoints {
		hashes = append(hashes, op.Hash.String())
		idxes = append(idxes, op.Index)
	}

	const q = `
		WITH outpoints AS (
			SELECT unnest($1::text[]) txid, unnest($2::int[])
		)
		UPDATE utxos SET reserved_until=NOW()
		WHERE (txid, index) IN (TABLE outpoints)
	`

	_, err := pg.FromContext(ctx).Exec(q, pg.Strings(hashes), pg.Uint32s(idxes))
	return errors.Wrap(err)
}
