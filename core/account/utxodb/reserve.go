// Package utxodb implements UTXO selection and reservation.
package utxodb

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/errors"
	chainlog "chain/log"
	"chain/protocol/bc"
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

		AccountID           string
		ControlProgramIndex uint64
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

func ReserveUTXO(ctx context.Context, txHash bc.Hash, pos uint32, clientToken *string, exp time.Time) (*UTXO, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "begin transaction for reserving utxos")
	}
	defer dbtx.Rollback(ctx)

	_, err = pg.Exec(ctx, `LOCK TABLE account_utxos IN ROW EXCLUSIVE MODE`)
	if err != nil {
		return nil, errors.Wrap(err, "acquire lock for reserving utxos")
	}

	const (
		reserveQ = `
			SELECT * FROM reserve_utxo($1, $2, $3, $4)
				AS (reservation_id INT, already_existed BOOLEAN, utxo_exists BOOLEAN)
		`
		utxosQ = `
			SELECT account_id, asset_id, amount, control_program_index, control_program
			FROM account_utxos
			WHERE reservation_id = $1 LIMIT 1
		`
	)

	var (
		reservationID  int32
		alreadyExisted bool
		utxoExists     bool
	)
	err = pg.QueryRow(ctx, reserveQ, txHash, pos, exp, clientToken).Scan(
		&reservationID,
		&alreadyExisted,
		&utxoExists,
	)
	if err != nil {
		return nil, errors.Wrap(err, "reserve utxo")
	}
	if reservationID <= 0 {
		if !utxoExists {
			return nil, pg.ErrUserInputNotFound
		}
		return nil, ErrReserved
	}

	var (
		accountID    string
		assetID      bc.AssetID
		amount       uint64
		programIndex uint64
		controlProg  []byte
	)

	err = pg.QueryRow(ctx, utxosQ, reservationID).Scan(&accountID, &assetID, &amount, &programIndex, &controlProg)
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "query reservation member")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "commit transaction for reserving utxo")
	}

	utxo := &UTXO{
		Outpoint: bc.Outpoint{
			Hash:  txHash,
			Index: pos,
		},
		AssetAmount: bc.AssetAmount{
			AssetID: assetID,
			Amount:  amount,
		},
		Script:              controlProg,
		AccountID:           accountID,
		ControlProgramIndex: programIndex,
	}

	return utxo, nil
}

func Reserve(ctx context.Context, sources []Source, exp time.Time) (u []*UTXO, c []Change, err error) {
	var reserved []*UTXO
	var change []Change
	var reservationIDs []int64

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
			pg.Exec(ctx, "SELECT cancel_reservations($1)", pq.Int64Array(reservationIDs)) // ignore errors
		}
	}()

	const (
		reserveQ = `
		SELECT * FROM reserve_utxos($1, $2, $3, $4, $5, $6, $7)
		    AS (reservation_id INT, already_existed BOOLEAN, existing_change BIGINT, amount BIGINT, insufficient BOOLEAN)
		`
		utxosQ = `
			SELECT a.tx_hash, a.index, a.amount, a.control_program_index, a.control_program
			FROM account_utxos a
			WHERE reservation_id = $1
		`
	)

	for _, source := range sources {
		var (
			txHash   sql.NullString
			outIndex sql.NullInt64

			reservationID  int64
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

		err = pg.ForQueryRows(ctx, pg.FromContext(ctx), utxosQ, reservationID, func(
			hash bc.Hash,
			index uint32,
			amount uint64,
			programIndex uint64,
			script []byte,
		) {
			utxo := UTXO{
				Outpoint:            bc.Outpoint{Hash: hash, Index: index},
				Script:              script,
				AssetAmount:         bc.AssetAmount{AssetID: source.AssetID, Amount: amount},
				AccountID:           source.AccountID,
				ControlProgramIndex: programIndex,
			}
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

// ExpireReservations is meant to be run as a goroutine. It loops,
// calling the expire_reservations() pl/pgsql function to
// remove expired reservations from the reservations table.
// It returns when its context is canceled.
func ExpireReservations(ctx context.Context, period time.Duration) {
	ticks := time.Tick(period)
	for {
		select {
		case <-ctx.Done():
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
