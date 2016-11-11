package utxodb

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/groupcache/singleflight"

	"chain/database/pg"
	"chain/errors"
	"chain/protocol/bc"
	"chain/sync/idempotency"
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

// UTXO describes an individual account UTXO.
type UTXO struct {
	bc.Outpoint
	bc.AssetAmount
	Script []byte

	AccountID           string
	ControlProgramIndex uint64
}

func (u *UTXO) source() Source {
	return Source{AssetID: u.AssetID, AccountID: u.AccountID}
}

// Source describes the criteria to use when selecting UTXOs.
type Source struct {
	AssetID   bc.AssetID
	AccountID string
}

// Reservation describes a reservation of a set of UTXOs belonging
// to a particular account. Reservations are immutable.
type Reservation struct {
	ID          uint64
	Source      Source
	UTXOs       []*UTXO
	Change      uint64
	Expiry      time.Time
	ClientToken *string
}

func NewReserver(db pg.DB) *Reserver {
	return &Reserver{
		db:           db,
		reservations: make(map[uint64]*Reservation),
		sources:      make(map[Source]*sourceReserver),
	}
}

// Reserver implements a UTXO reserver that stores reservations
// in-memory. It relies on the account_utxos table for the source of
// truth of valid UTXOs but tracks which of those UTXOs are reserved
// in-memory.
//
// To reduce latency and prevent deadlock, no two mutexes (either on
// Reserver or sourceReserver) should be held at the same time
//
// Reserver ensures idempotency of reservations until the reservation
// expiration.
type Reserver struct {
	db                pg.DB
	nextReservationID uint64
	idempotency       idempotency.Group

	reservationsMu sync.Mutex
	reservations   map[uint64]*Reservation

	sourcesMu sync.Mutex
	sources   map[Source]*sourceReserver
}

// Reserve selects and reserves UTXOs according to the critera provided
// in source. The resulting reservation expires at exp.
func (re *Reserver) Reserve(ctx context.Context, source Source, amount uint64, clientToken *string, exp time.Time) (*Reservation, error) {
	if clientToken == nil {
		return re.reserve(ctx, source, amount, clientToken, exp)
	}

	untypedRes, err := re.idempotency.Once(*clientToken, func() (interface{}, error) {
		return re.reserve(ctx, source, amount, clientToken, exp)
	})
	return untypedRes.(*Reservation), err
}

func (re *Reserver) reserve(ctx context.Context, source Source, amount uint64, clientToken *string, exp time.Time) (res *Reservation, err error) {
	sourceReserver := re.source(source)

	// Find the set of UTXOs that match this source.
	utxos, err := sourceReserver.findMatchingUTXOs(ctx)
	if err != nil {
		return nil, err
	}

	// Try to reserve the right amount.
	rid := atomic.AddUint64(&re.nextReservationID, 1)
	reserved, total, err := sourceReserver.reserve(rid, amount, utxos)
	if err != nil {
		return nil, err
	}

	res = &Reservation{
		ID:          rid,
		Source:      source,
		UTXOs:       reserved,
		Expiry:      exp,
		ClientToken: clientToken,
	}

	// Save the successful reservation.
	re.reservationsMu.Lock()
	defer re.reservationsMu.Unlock()
	re.reservations[rid] = res

	// Make change if necessary
	if total > amount {
		res.Change = total - amount
	}
	return res, nil
}

// ReserveUTXO reserves a specific UTXO for spending. The resulting
// reservation expires at exp.
func (re *Reserver) ReserveUTXO(ctx context.Context, out bc.Outpoint, clientToken *string, exp time.Time) (*Reservation, error) {
	if clientToken == nil {
		return re.reserveUTXO(ctx, out, exp, nil)
	}

	untypedRes, err := re.idempotency.Once(*clientToken, func() (interface{}, error) {
		return re.reserveUTXO(ctx, out, exp, clientToken)
	})
	return untypedRes.(*Reservation), err
}

func (re *Reserver) reserveUTXO(ctx context.Context, out bc.Outpoint, exp time.Time, clientToken *string) (*Reservation, error) {
	utxo, err := findSpecificUTXO(ctx, re.db, out)
	if err != nil {
		return nil, err
	}

	rid := atomic.AddUint64(&re.nextReservationID, 1)
	err = re.source(utxo.source()).reserveUTXO(rid, utxo)
	if err != nil {
		return nil, err
	}

	res := &Reservation{
		ID:          rid,
		Source:      utxo.source(),
		UTXOs:       []*UTXO{utxo},
		Expiry:      exp,
		ClientToken: clientToken,
	}
	re.reservationsMu.Lock()
	re.reservations[rid] = res
	re.reservationsMu.Unlock()
	return res, nil
}

// Cancel makes a best-effort attempt at canceling the reservation with
// the provided ID.
func (re *Reserver) Cancel(ctx context.Context, rid uint64) error {
	re.reservationsMu.Lock()
	res, ok := re.reservations[rid]
	delete(re.reservations, rid)
	re.reservationsMu.Unlock()
	if !ok {
		return fmt.Errorf("couldn't find reservation %d", rid)
	}
	re.source(res.Source).cancel(res)
	if res.ClientToken != nil {
		re.idempotency.Forget(*res.ClientToken)
	}
	return nil
}

// ExpireReservations cleans up all reservations that have expired,
// making their UTXOs available for reservation again.
func (re *Reserver) ExpireReservations(ctx context.Context) error {
	// Remove records of any reservations that have expired.
	now := time.Now()
	var canceled []*Reservation
	re.reservationsMu.Lock()
	for rid, res := range re.reservations {
		if res.Expiry.Before(now) {
			canceled = append(canceled, res)
			delete(re.reservations, rid)
		}
	}
	re.reservationsMu.Unlock()

	// If we removed any expired reservations, update the corresponding
	// source reservers.
	for _, res := range canceled {
		re.source(res.Source).cancel(res)
		if res.ClientToken != nil {
			re.idempotency.Forget(*res.ClientToken)
		}
	}

	// TODO(jackson): Cleanup any source reservers that don't have
	// anything reserved. It'll be a little tricky because of our
	// locking scheme.
	return nil
}

func (re *Reserver) source(src Source) *sourceReserver {
	re.sourcesMu.Lock()
	defer re.sourcesMu.Unlock()

	sr, ok := re.sources[src]
	if ok {
		return sr
	}

	sr = &sourceReserver{
		db:       re.db,
		source:   src,
		reserved: make(map[bc.Outpoint]uint64),
	}
	re.sources[src] = sr
	return sr
}

type sourceReserver struct {
	db     pg.DB
	source Source
	group  singleflight.Group

	mu       sync.Mutex
	reserved map[bc.Outpoint]uint64
}

func (sr *sourceReserver) findMatchingUTXOs(ctx context.Context) ([]*UTXO, error) {
	srcID := fmt.Sprintf("%s-%s", sr.source.AssetID, sr.source.AccountID)
	untypedUTXOs, err := sr.group.Do(srcID, func() (interface{}, error) {
		return findMatchingUTXOs(ctx, sr.db, sr.source)
	})
	return untypedUTXOs.([]*UTXO), err
}

func (sr *sourceReserver) reserve(rid uint64, amount uint64, utxos []*UTXO) ([]*UTXO, uint64, error) {
	var reserved, unavailable uint64
	var reservedUTXOs []*UTXO

	sr.mu.Lock()
	defer sr.mu.Unlock()
	for _, utxo := range utxos {
		// If the UTXO is already reserved, skip it.
		if _, ok := sr.reserved[utxo.Outpoint]; ok {
			unavailable += utxo.Amount
			continue
		}

		// This UTXO is available for the taking.
		reserved += utxo.Amount
		reservedUTXOs = append(reservedUTXOs, utxo)
		if reserved >= amount {
			break
		}
	}

	if reserved+unavailable < amount {
		// Even if everything was available, this account wouldn't have
		// enough to satisfy the request.
		return nil, 0, ErrInsufficient
	}
	if reserved < amount {
		// The account has enough for the request, but some is tied up in
		// other reservations.
		return nil, 0, ErrReserved
	}

	// We've found enough to satisfy the request.
	for _, utxo := range reservedUTXOs {
		sr.reserved[utxo.Outpoint] = rid
	}
	return reservedUTXOs, reserved, nil
}

func (sr *sourceReserver) reserveUTXO(rid uint64, utxo *UTXO) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	_, isReserved := sr.reserved[utxo.Outpoint]
	if isReserved {
		return ErrReserved
	}

	sr.reserved[utxo.Outpoint] = rid
	return nil
}

func (sr *sourceReserver) cancel(res *Reservation) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	for _, utxo := range res.UTXOs {
		delete(sr.reserved, utxo.Outpoint)
	}
}

func findMatchingUTXOs(ctx context.Context, db pg.DB, source Source) ([]*UTXO, error) {
	const q = `
		SELECT tx_hash, index, amount, control_program_index, control_program
		FROM account_utxos
		WHERE account_id = $1 AND asset_id = $2
	`
	var utxos []*UTXO
	err := pg.ForQueryRows(ctx, db, q, source.AccountID, source.AssetID,
		func(txHash bc.Hash, index uint32, amount uint64, cpIndex uint64, controlProg []byte) {
			utxos = append(utxos, &UTXO{
				Outpoint: bc.Outpoint{
					Hash:  txHash,
					Index: index,
				},
				AssetAmount: bc.AssetAmount{
					Amount:  amount,
					AssetID: source.AssetID,
				},
				Script:              controlProg,
				AccountID:           source.AccountID,
				ControlProgramIndex: cpIndex,
			})
		})
	// TODO(jackson): This has the potential to be a large number of UTXOs.
	// If we need to, we can cache UTXOs or at least avoid reading UTXOs once
	// we've found enough to satisfy the reservation.
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return utxos, nil
}

func findSpecificUTXO(ctx context.Context, db pg.DB, out bc.Outpoint) (*UTXO, error) {
	const q = `
		SELECT account_id, asset_id, amount, control_program_index, control_program
		FROM account_utxos
		WHERE tx_hash = $1 AND index = $2
	`
	u := new(UTXO)
	err := db.QueryRow(ctx, q, out.Hash, out.Index).Scan(&u.AccountID, &u.AssetID, &u.Amount, &u.ControlProgramIndex, &u.Script)
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	} else if err != nil {
		return nil, errors.Wrap(err)
	}
	u.Outpoint = out
	return u, nil
}
