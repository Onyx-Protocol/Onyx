package utxodb

import (
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	chainlog "chain/log"
	"chain/metrics"
	"chain/net/trace/span"
)

var (
	// ErrInsufficient indicates the account doesn't contain enough
	// units of the requested asset to satisfy the reservation.
	// New units must be deposited into the account in order to
	// satisfy the request; change will not be sufficient.
	ErrInsufficient = errors.New("reservation found insufficient funds")

	// ErrReserved indicates that a reservation could not be
	// satisfied because some of the outputs were already reserved.
	// When those reservations are finalized into a transaction
	// (and no other transaction spends funds from the account),
	// new change outputs will be created
	// in sufficient amounts to satisfy the request.
	ErrReserved = errors.New("reservation found outputs already reserved")
)

type (
	key struct {
		AccountID string
		AssetID   bc.AssetID
	}

	// TODO(kr): see if we can avoid storing
	// AccountID and AssetID in UTXO

	// TODO(kr): try interning strings in UTXO

	UTXO struct {
		bc.Outpoint
		bc.AssetAmount

		AccountID string
		AddrIndex [2]uint32
	}

	Receiver struct {
		ManagerNodeID string   `json:"manager_node_id"`
		AccountID     string   `json:"account_id"`
		AddrIndex     []uint32 `json:"address_index"`
	}

	// Change represents reserved units beyond what was asked for.
	// Total reservation is for Amount+Source.Amount.
	Change struct {
		Source Source
		Amount uint64
	}

	Source struct {
		AssetID     bc.AssetID `json:"asset_id"`
		AccountID   string     `json:"account_id"`
		TxID        string     `json:"transaction_id"` // TODO(bobg): remove this, it's unused
		Amount      uint64
		ClientToken *string `json:"client_token"`
	}
)

func Reserve(ctx context.Context, sources []Source, ttl time.Duration) (u []*UTXO, c []Change, err error) {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var reserved []*UTXO
	var change []Change
	var reservationIDs []int32

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "begin transaction for reserving utxos")
	}
	defer dbtx.Rollback(ctx)

	db := pg.FromContext(ctx)

	_, err = db.Exec(ctx, `LOCK TABLE account_utxos IN ROW EXCLUSIVE MODE`)
	if err != nil {
		return nil, nil, errors.Wrap(err, "acquire lock for reserving utxos")
	}

	defer func() {
		if err != nil {
			db.Exec(ctx, "SELECT cancel_reservations($1)", pg.Int32s(reservationIDs)) // ignore errors
		}
	}()

	now := time.Now().UTC()
	exp := now.Add(ttl)

	const (
		reserveQ = `
		SELECT * FROM reserve_utxos($1, $2, $3, $4, $5)
		    AS (reservation_id INT, already_existed BOOLEAN, existing_change BIGINT, amount BIGINT, insufficient BOOLEAN)
		`
		utxosQ = `
		SELECT tx_hash, index, amount, key_index(addr_index)
		    FROM account_utxos
		    WHERE reservation_id = $1
		`
	)

	for _, source := range sources {
		var (
			reservationID  int32
			alreadyExisted bool
			existingChange uint64
			reservedAmount uint64
			insufficient   bool
		)

		// Create a reservation row and reserve the utxos. If this reservation
		// has alredy been processed in a previous request:
		//  * the existing reservation ID will be returned
		//  * already_existed will be TRUE
		//  * existing_change will be the change value for the existing
		//    reservation row.
		err = db.QueryRow(ctx, reserveQ, source.AssetID, source.AccountID, source.Amount, exp, source.ClientToken).Scan(
			&reservationID,
			&alreadyExisted,
			&existingChange,
			&reservedAmount,
			&insufficient,
		)
		if err != nil {
			return nil, nil, errors.Wrap(err, "reserve utxos")
		}
		if reservationID <= 0 {
			if insufficient {
				return nil, nil, ErrInsufficient
			}
			return nil, nil, ErrReserved
		}

		reservationIDs = append(reservationIDs, reservationID)

		if alreadyExisted && existingChange > 0 {
			// This reservation already exists from a previous request
			change = append(change, Change{source, existingChange})
		} else if reservedAmount > source.Amount {
			change = append(change, Change{source, reservedAmount - source.Amount})
		}

		rows, err := db.Query(ctx, utxosQ, reservationID)
		if err != nil {
			return nil, nil, errors.Wrap(err, "reservation member query")
		}
		defer rows.Close()

		for rows.Next() {
			// TODO(bobg): Sort utxos from largest amount to smallest, which
			// might allow us to satisfy source.Amount with fewer utxos,
			// unreserving some and making less change.
			var addrIndex []uint32
			utxo := UTXO{
				AssetAmount: bc.AssetAmount{
					AssetID: source.AssetID,
				},
				AccountID: source.AccountID,
			}
			err = rows.Scan(&utxo.Hash, &utxo.Index, &utxo.Amount, (*pg.Uint32s)(&addrIndex))
			if err != nil {
				return nil, nil, errors.Wrap(err, "reservation member row scan")
			}
			copy(utxo.AddrIndex[:], addrIndex)
			reserved = append(reserved, &utxo)
		}
		if err = rows.Err(); err != nil {
			return nil, nil, errors.Wrap(err, "end reservation member row scan loop")
		}
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "commit transaction for reserving utxos")
	}

	return reserved, change, err
}

// Cancel cancels the given reservations, if they still exist.
// If any do not exist (if they've already been consumed
// or canceled), it silently ignores them.
func Cancel(ctx context.Context, outpoints []bc.Outpoint) error {
	txHashes := make([]bc.Hash, 0, len(outpoints))
	indexes := make([]uint32, 0, len(outpoints))
	for _, outpoint := range outpoints {
		txHashes = append(txHashes, outpoint.Hash)
		indexes = append(indexes, outpoint.Index)
	}

	const query = `
		WITH reservation_ids AS (
		    SELECT DISTINCT reservation_id FROM utxos
		        WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::bigint[]))
		)
		SELECT cancel_reservation(reservation_id) FROM reservation_ids
	`

	_, err := pg.FromContext(ctx).Exec(ctx, query, txHashes, indexes)
	return err
}

// ExpireReservations is meant to be run as a goroutine. It loops
// forever, calling the expire_reservations() pl/pgsql function to
// remove expired reservations from the reservations table.
func ExpireReservations(ctx context.Context, period time.Duration) {
	for range time.Tick(period) {
		err := func() error {
			dbtx, ctx, err := pg.Begin(ctx)
			if err != nil {
				return err
			}
			defer dbtx.Rollback(ctx)

			db := pg.FromContext(ctx)

			_, err = db.Exec(ctx, `LOCK TABLE account_utxos IN EXCLUSIVE MODE`)
			if err != nil {
				return err
			}

			_, err = db.Exec(ctx, `SELECT expire_reservations()`)
			if err != nil {
				return err
			}

			return dbtx.Commit(ctx)
		}()
		if err != nil {
			chainlog.Error(ctx, err)
		}
	}
}
