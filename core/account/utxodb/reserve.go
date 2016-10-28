package utxodb

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

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

// Source describes the criteria to use when selecting UTXOs.
type Source struct {
	bc.AssetAmount
	AccountID   string  `json:"account_id"`
	ClientToken *string `json:"client_token"`
}

// Reservation describes a reservation of a set of UTXOs belonging
// to a particular account. Reservations are immutable.
type Reservation struct {
	ID          uint64
	AccountID   string
	UTXOs       []*UTXO
	Change      uint64
	Expiry      time.Time
	ClientToken *string
}

func NewReserver(db pg.DB) *Reserver {
	return &Reserver{
		db:           db,
		reservations: make(map[uint64]*Reservation),
		accounts:     make(map[string]*accountReserver),
	}
}

// Reserver implements a UTXO reserver that stores reservations
// in-memory. It relies on the account_utxos table for the source of
// truth of valid UTXOs but tracks which of those UTXOs are reserved
// in-memory.
//
// To reduce latency and prevent deadlock, no two mutexes (either on
// Reserver or accountReserver) should be held at the same time
//
// Reserver ensures idempotency of reservations until the reservation
// expiration.
type Reserver struct {
	db                pg.DB
	nextReservationID uint64
	idempotency       idempotency.Group

	reservationsMu sync.Mutex
	reservations   map[uint64]*Reservation

	accountsMu sync.Mutex
	accounts   map[string]*accountReserver
}

// Reserve selects and reserves UTXOs according to the critera provided
// in source. The resulting reservation expires at exp.
func (re *Reserver) Reserve(ctx context.Context, source Source, exp time.Time) (*Reservation, error) {
	if source.ClientToken == nil {
		return re.reserve(ctx, source, exp)
	}

	untypedRes, err := re.idempotency.Once(*source.ClientToken, func() (interface{}, error) {
		return re.reserve(ctx, source, exp)
	})
	return untypedRes.(*Reservation), err
}

func (re *Reserver) reserve(ctx context.Context, source Source, exp time.Time) (res *Reservation, err error) {
	// Find the set of UTXOs that match this source.
	utxos, err := re.findMatchingUTXOs(ctx, source)
	if err != nil {
		return nil, err
	}

	// Try to reserve the right amount.
	rid := atomic.AddUint64(&re.nextReservationID, 1)
	reserved, total, err := re.account(source.AccountID).reserve(rid, source, utxos)
	if err != nil {
		return nil, err
	}

	res = &Reservation{
		ID:          rid,
		AccountID:   source.AccountID,
		UTXOs:       reserved,
		Expiry:      exp,
		ClientToken: source.ClientToken,
	}

	// Save the successful reservation.
	re.reservationsMu.Lock()
	defer re.reservationsMu.Unlock()
	re.reservations[rid] = res

	// Make change if necessary
	if total > source.Amount {
		res.Change = total - source.Amount
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
	utxo, err := re.findSpecificUTXO(ctx, out)
	if err != nil {
		return nil, err
	}

	rid := atomic.AddUint64(&re.nextReservationID, 1)
	err = re.account(utxo.AccountID).reserveUTXO(rid, utxo)
	if err != nil {
		return nil, err
	}

	res := &Reservation{
		ID:          rid,
		AccountID:   utxo.AccountID,
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
	re.account(res.AccountID).cancel(res)
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
	// acount reservers.
	for _, res := range canceled {
		re.account(res.AccountID).cancel(res)
		if res.ClientToken != nil {
			re.idempotency.Forget(*res.ClientToken)
		}
	}

	// Cleanup any account reservers that don't have anything reserved.
	re.accountsMu.Lock()
	for accID, ar := range re.accounts {
		if len(ar.reserved) == 0 {
			delete(re.accounts, accID)
		}
	}
	re.accountsMu.Unlock()
	return nil
}

func (re *Reserver) findMatchingUTXOs(ctx context.Context, source Source) ([]*UTXO, error) {
	const q = `
		SELECT tx_hash, index, amount, control_program_index, control_program, confirmed_in
		FROM account_utxos a
		WHERE account_id = $1 AND asset_id = $2
	`
	var utxos []*UTXO
	err := pg.ForQueryRows(ctx, re.db, q, source.AccountID, source.AssetID,
		func(txHash bc.Hash, index uint32, amount uint64, cpIndex uint64, controlProg []byte, confirmedIn *uint64) {
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
	if err != nil {
		return nil, errors.Wrap(err)
	}
	for i := range utxos {
		j := rand.Intn(i + 1)
		utxos[i], utxos[j] = utxos[j], utxos[i]
	}
	return utxos, nil
}

func (re *Reserver) findSpecificUTXO(ctx context.Context, out bc.Outpoint) (*UTXO, error) {
	const q = `
		SELECT account_id, asset_id, amount, control_program_index, control_program
		FROM account_utxos
		WHERE tx_hash = $1 AND index = $2
	`
	u := new(UTXO)
	err := re.db.QueryRow(ctx, q, out.Hash, out.Index).Scan(&u.AccountID, &u.AssetID, &u.Amount, &u.ControlProgramIndex, &u.Script)
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	} else if err != nil {
		return nil, errors.Wrap(err)
	}
	u.Outpoint = out
	return u, nil
}

func (re *Reserver) account(accID string) *accountReserver {
	re.accountsMu.Lock()
	defer re.accountsMu.Unlock()

	ar, ok := re.accounts[accID]
	if ok {
		return ar
	}

	ar = &accountReserver{
		reserved: make(map[bc.Outpoint]uint64),
	}
	re.accounts[accID] = ar
	return ar
}

type accountReserver struct {
	mu       sync.Mutex
	reserved map[bc.Outpoint]uint64
}

func (ar *accountReserver) reserve(rid uint64, src Source, utxos []*UTXO) ([]*UTXO, uint64, error) {
	var reserved, unavailable uint64
	var reservedUTXOs []*UTXO

	ar.mu.Lock()
	defer ar.mu.Unlock()
	for _, utxo := range utxos {
		// If the UTXO is already reserved, skip it.
		if _, ok := ar.reserved[utxo.Outpoint]; ok {
			unavailable += utxo.Amount
			continue
		}

		// This UTXO is available for the taking.
		reserved += utxo.Amount
		reservedUTXOs = append(reservedUTXOs, utxo)
		if reserved >= src.Amount {
			break
		}
	}

	if reserved+unavailable < src.Amount {
		// Even if everything was available, this account wouldn't have
		// enough to satisfy the request.
		return nil, 0, ErrInsufficient
	}
	if reserved < src.Amount {
		// The account has enough for the request, but some is tied up in
		// other reservations.
		return nil, 0, ErrReserved
	}

	// We've found enough to satisfy the request.
	for _, utxo := range reservedUTXOs {
		ar.reserved[utxo.Outpoint] = rid
	}
	return reservedUTXOs, reserved, nil
}

func (ar *accountReserver) reserveUTXO(rid uint64, utxo *UTXO) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	_, isReserved := ar.reserved[utxo.Outpoint]
	if isReserved {
		return ErrReserved
	}

	ar.reserved[utxo.Outpoint] = rid
	return nil
}

func (ar *accountReserver) cancel(res *Reservation) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	for _, utxo := range res.UTXOs {
		delete(ar.reserved, utxo.Outpoint)
	}
}
