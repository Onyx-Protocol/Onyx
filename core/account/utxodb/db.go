// Package utxodb implements UTXO selection and reservation.
package utxodb

import (
	"context"
	stdsql "database/sql"
	"time"

	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
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

type DBReserver struct {
	DB *sql.DB
}

type UTXO struct {
	bc.Outpoint
	bc.AssetAmount
	Script []byte

	AccountID           string
	ControlProgramIndex uint64
}

type Source struct {
	bc.AssetAmount
	AccountID   string `json:"account_id"`
	TxHash      *bc.Hash
	OutputIndex *uint32
	ClientToken *string `json:"client_token"`
}

func (res *DBReserver) ReserveUTXO(ctx context.Context, txHash bc.Hash, pos uint32, clientToken *string, exp time.Time) (int32, *UTXO, error) {
	dbtx, err := res.DB.Begin(ctx)
	if err != nil {
		return 0, nil, errors.Wrap(err, "begin transaction for reserving utxo")
	}
	defer dbtx.Rollback(ctx)

	// Lock in ROW EXCLUSIVE mode because we believe the row affected
	// here can't be affected by other threads concurrently. (That may
	// be too optimistic. If it's just been spent [by another Core,
	// presumably] it might get deleted during utxo indexing while we're
	// trying to reserve it.)
	_, err = dbtx.Exec(ctx, `LOCK TABLE account_utxos IN ROW EXCLUSIVE MODE`)
	if err != nil {
		return 0, nil, errors.Wrap(err, "acquire lock for reserving utxos")
	}

	const (
		reserveQ = `
			SELECT * FROM reserve_utxo($1, $2, $3, $4)
				AS (reservation_id INT, already_existed BOOLEAN, utxo_exists BOOLEAN)
		`
		utxoQ = `
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

	err = dbtx.QueryRow(ctx, utxoQ, reservationID).Scan(&accountID, &assetID, &amount, &programIndex, &controlProg)
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

	return reservationID, utxo, nil
}

// Reserve reserves account UTXOs to cover the provided sources. If
// UTXOs are successfully reserved, it's the responsbility of the
// caller to cancel them if an error occurs.
func (res *DBReserver) Reserve(ctx context.Context, source Source, exp time.Time) (reservationID int32, reserved []*UTXO, change uint64, err error) {
	dbtx, err := res.DB.Begin(ctx)
	if err != nil {
		return 0, nil, 0, errors.Wrap(err, "begin transaction for reserving utxos")
	}
	defer dbtx.Rollback(ctx)

	// Lock in ROW EXCLUSIVE mode because we believe the rows affected
	// here can't be affected by other threads concurrently. (That may
	// be too optimistic. If they've just been spent [by another Core,
	// presumably] they might get deleted during utxo indexing while
	// we're trying to reserve them.)
	_, err = dbtx.Exec(ctx, `LOCK TABLE account_utxos IN ROW EXCLUSIVE MODE`)
	if err != nil {
		return 0, nil, 0, errors.Wrap(err, "acquire lock for reserving utxos")
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

	var (
		txHash   stdsql.NullString
		outIndex stdsql.NullInt64
	)

	if source.TxHash != nil {
		txHash.Valid = true
		txHash.String = source.TxHash.String()
	}

	if source.OutputIndex != nil {
		outIndex.Valid = true
		outIndex.Int64 = int64(*source.OutputIndex)
	}

	var (
		alreadyExisted bool
		existingChange uint64
		reservedAmount uint64
		insufficient   bool
	)

	// Create a reservation row and reserve the utxos. If this reservation
	// has already been processed in a previous request:
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
		return 0, nil, 0, errors.Wrap(err, "reserve utxos")
	}
	if reservationID <= 0 {
		if insufficient {
			return 0, nil, 0, ErrInsufficient
		}
		return 0, nil, 0, ErrReserved
	}

	if alreadyExisted && existingChange > 0 {
		// This reservation already exists from a previous request
		change = existingChange
	} else if reservedAmount > source.Amount {
		change = reservedAmount - source.Amount
	}

	err = pg.ForQueryRows(ctx, dbtx, utxosQ, reservationID,
		func(hash bc.Hash, index uint32, amount uint64, programIndex uint64, script []byte) {
			utxo := UTXO{
				Outpoint:            bc.Outpoint{Hash: hash, Index: index},
				Script:              script,
				AssetAmount:         bc.AssetAmount{AssetID: source.AssetID, Amount: amount},
				AccountID:           source.AccountID,
				ControlProgramIndex: programIndex,
			}
			reserved = append(reserved, &utxo)
		},
	)
	if err != nil {
		return reservationID, nil, 0, errors.Wrap(err, "query reservation members")
	}
	err = dbtx.Commit(ctx)
	if err != nil {
		return reservationID, nil, 0, errors.Wrap(err, "commit transaction for reserving utxos")
	}
	return reservationID, reserved, change, err
}

// Cancel cancels the given reservation if possible.
// If it doesn't exist (if it's already been consumed
// or canceled), it is silently ignored.
func (res *DBReserver) Cancel(ctx context.Context, rid int32) error {
	dbtx, err := res.DB.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "begin transaction for canceling utxo reservation")
	}
	defer dbtx.Rollback(ctx)

	// Lock in ROW EXCLUSIVE mode because we believe the row affected
	// here can't be affected by other threads concurrently. (That may
	// be too optimistic. If an underlying utxo has just been spent [by
	// another Core, presumably] it might get deleted at the same time
	// we're trying to null out its reservation_id as a result of
	// canceling the reservation.)
	_, err = dbtx.Exec(ctx, `LOCK TABLE account_utxos IN ROW EXCLUSIVE MODE`)
	if err != nil {
		return errors.Wrap(err, "locking table for canceling utxo reservation")
	}

	_, err = dbtx.Exec(ctx, "SELECT cancel_reservation($1)", rid)
	if err != nil {
		return errors.Wrap(err, "canceling utxo reservation")
	}

	return dbtx.Commit(ctx)
}

func (res *DBReserver) ExpireReservations(ctx context.Context) error {
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
}
