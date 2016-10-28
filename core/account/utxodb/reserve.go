// Package utxodb implements UTXO selection and reservation.
package utxodb

import (
	"context"
	stdsql "database/sql"
	"time"

	"chain/database/pg"
	"chain/database/sql"
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
	Reserver struct {
		DB *sql.DB
	}

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

func (res *Reserver) ReserveUTXO(ctx context.Context, txHash bc.Hash, pos uint32, clientToken *string, exp time.Time) (reservationID int32, utxo *UTXO, err error) {
	dbtx, err := res.DB.Begin(ctx)
	if err != nil {
		return 0, nil, errors.Wrap(err, "begin transaction for reserving utxos")
	}
	defer dbtx.Rollback(ctx)

	_, err = dbtx.Exec(ctx, `LOCK TABLE account_utxos IN ROW EXCLUSIVE MODE`)
	if err != nil {
		return 0, nil, errors.Wrap(err, "acquire lock for reserving utxos")
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
		alreadyExisted bool
		utxoExists     bool
	)
	err = dbtx.QueryRow(ctx, reserveQ, txHash, pos, exp, clientToken).Scan(
		&reservationID,
		&alreadyExisted,
		&utxoExists,
	)
	if err != nil {
		return 0, nil, errors.Wrap(err, "reserve utxo")
	}
	if reservationID <= 0 {
		if !utxoExists {
			return 0, nil, pg.ErrUserInputNotFound
		}
		return 0, nil, ErrReserved
	}

	var (
		accountID    string
		assetID      bc.AssetID
		amount       uint64
		programIndex uint64
		controlProg  []byte
	)

	err = dbtx.QueryRow(ctx, utxosQ, reservationID).Scan(&accountID, &assetID, &amount, &programIndex, &controlProg)
	if err == stdsql.ErrNoRows {
		return 0, nil, pg.ErrUserInputNotFound
	}
	if err != nil {
		return 0, nil, errors.Wrap(err, "query reservation member")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return 0, nil, errors.Wrap(err, "commit transaction for reserving utxo")
	}

	utxo = &UTXO{
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

	return reservationID, utxo, nil
}

// Reserve reserves account UTXOs to cover the provided sources. If
// UTXOs are successfully reserved, it's the responsbility of the
// caller to cancel them if an error occurs.
func (res *Reserver) Reserve(ctx context.Context, sources []Source, exp time.Time) (reservationIDs []int32, u []*UTXO, c []Change, err error) {
	var reserved []*UTXO
	var change []Change

	dbtx, err := res.DB.Begin(ctx)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "begin transaction for reserving utxos")
	}
	defer dbtx.Rollback(ctx)

	_, err = dbtx.Exec(ctx, `LOCK TABLE account_utxos IN ROW EXCLUSIVE MODE`)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "acquire lock for reserving utxos")
	}
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

	// TODO(kr): make sources not be a list;
	// we only ever call Reserve with a single item.
	for _, source := range sources {
		var (
			txHash   stdsql.NullString
			outIndex stdsql.NullInt64

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
		err = dbtx.QueryRow(ctx, reserveQ, source.AssetID, source.AccountID, txHash, outIndex, source.Amount, exp, source.ClientToken).Scan(
			&reservationID,
			&alreadyExisted,
			&existingChange,
			&reservedAmount,
			&insufficient,
		)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "reserve utxos")
		}
		if reservationID <= 0 {
			if insufficient {
				return nil, nil, nil, ErrInsufficient
			}
			return nil, nil, nil, ErrReserved
		}

		reservationIDs = append(reservationIDs, reservationID)

		if alreadyExisted && existingChange > 0 {
			// This reservation already exists from a previous request
			change = append(change, Change{source, existingChange})
		} else if reservedAmount > source.Amount {
			change = append(change, Change{source, reservedAmount - source.Amount})
		}

		err = pg.ForQueryRows(ctx, dbtx, utxosQ, reservationID, func(
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
			return nil, nil, nil, errors.Wrap(err, "query reservation members")
		}
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "commit transaction for reserving utxos")
	}

	return reservationIDs, reserved, change, err
}

// Cancel cancels the given reservation if possible.
// If it doesn't exist (if it's already been consumed
// or canceled), it is silently ignored.
func (res *Reserver) Cancel(ctx context.Context, rid int32) error {
	_, err := res.DB.Exec(ctx, "SELECT cancel_reservation($1)", rid)
	return err
}

// ExpireReservations is meant to be run as a goroutine. It loops,
// calling the expire_reservations() pl/pgsql function to
// remove expired reservations from the reservations table.
// It returns when its context is canceled.
func (res *Reserver) ExpireReservations(ctx context.Context, period time.Duration) {
	ticks := time.Tick(period)
	for {
		select {
		case <-ctx.Done():
			chainlog.Messagef(ctx, "Deposed, ExpireReservations exiting")
			return
		case <-ticks:
			err := func() error {
				dbtx, err := res.DB.Begin(ctx)
				if err != nil {
					return err
				}
				defer dbtx.Rollback(ctx)

				_, err = dbtx.Exec(ctx, `LOCK TABLE account_utxos IN EXCLUSIVE MODE`)
				if err != nil {
					return err
				}

				_, err = dbtx.Exec(ctx, `SELECT expire_reservations()`)
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
