package utxodb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/database/pg"
	"chain/errors"
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
	UTXO struct {
		bc.Outpoint
		bc.AssetAmount
		Script []byte

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
		TxHash      *bc.Hash
		OutputIndex *uint32
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

	_, err = pg.Exec(ctx, `LOCK TABLE account_utxos IN ROW EXCLUSIVE MODE`)
	if err != nil {
		return nil, nil, errors.Wrap(err, "acquire lock for reserving utxos")
	}

	defer func() {
		if err != nil {
			pg.Exec(ctx, "SELECT cancel_reservations($1)", pg.Int32s(reservationIDs)) // ignore errors
		}
	}()

	now := time.Now().UTC()
	exp := now.Add(ttl)

	const (
		reserveQ = `
		SELECT * FROM reserve_utxos($1, $2, $3, $4, $5, $6, $7)
		    AS (reservation_id INT, already_existed BOOLEAN, existing_change BIGINT, amount BIGINT, insufficient BOOLEAN)
		`
		utxosQ = `
			SELECT a.tx_hash, a.index, a.amount, key_index(a.addr_index), a.script
			FROM account_utxos a
			WHERE reservation_id = $1
		`
	)

	for _, source := range sources {
		var (
			txHash   sql.NullString
			outIndex sql.NullInt64

			reservationID  int32
			alreadyExisted bool
			existingChange uint64
			reservedAmount uint64
			insufficient   bool
		)

		if source.TxHash != nil {
			txHash.Valid = true
			txHash.String = source.TxHash.String()
		}

		if source.OutputIndex != nil {
			outIndex.Valid = true
			outIndex.Int64 = int64(*source.OutputIndex)
		}

		// Create a reservation row and reserve the utxos. If this reservation
		// has alredy been processed in a previous request:
		//  * the existing reservation ID will be returned
		//  * already_existed will be TRUE
		//  * existing_change will be the change value for the existing
		//    reservation row.
		err = pg.QueryRow(ctx, reserveQ, source.AssetID, source.AccountID, txHash, outIndex, source.Amount, exp, source.ClientToken).Scan(
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

		err = pg.ForQueryRows(ctx, utxosQ, reservationID, func(
			hash bc.Hash,
			index uint32,
			amount uint64,
			addrIndex pg.Uint32s,
			script []byte,
		) {
			utxo := UTXO{
				Outpoint:    bc.Outpoint{Hash: hash, Index: index},
				Script:      script,
				AssetAmount: bc.AssetAmount{AssetID: source.AssetID, Amount: amount},
				AccountID:   source.AccountID,
			}
			copy(utxo.AddrIndex[:], addrIndex)
			reserved = append(reserved, &utxo)
		})
		if err != nil {
			return nil, nil, errors.Wrap(err, "query reservation members")
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
	txHashes := make([]string, 0, len(outpoints))
	indexes := make([]int32, 0, len(outpoints))
	for _, outpoint := range outpoints {
		txHashes = append(txHashes, outpoint.Hash.String())
		indexes = append(indexes, int32(outpoint.Index))
	}

	const query = `
		WITH reservation_ids AS (
		    SELECT DISTINCT reservation_id FROM account_utxos
		        WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::bigint[]))
		)
		SELECT cancel_reservation(reservation_id) FROM reservation_ids
	`

	_, err := pg.Exec(ctx, query, pg.Strings(txHashes), pg.Int32s(indexes))
	return err
}

// ExpireReservations is meant to be run as a goroutine. It loops
// forever, calling the expire_reservations() pl/pgsql function to
// remove expired reservations from the reservations table.
func ExpireReservations(ctx context.Context, period time.Duration, deposed <-chan struct{}) {
	ticks := time.Tick(period)
	for {
		select {
		case <-deposed:
			chainlog.Messagef(ctx, "Deposed, ExpireReservations exiting")
			return
		case <-ticks:
			err := func() error {
				dbtx, ctx, err := pg.Begin(ctx)
				if err != nil {
					return err
				}
				defer dbtx.Rollback(ctx)

				_, err = pg.Exec(ctx, `LOCK TABLE account_utxos IN EXCLUSIVE MODE`)
				if err != nil {
					return err
				}

				_, err = pg.Exec(ctx, `SELECT expire_reservations()`)
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
}
