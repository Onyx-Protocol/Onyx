package account

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"chain/core/pin"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
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

// utxo describes an individual account utxo.
type utxo struct {
	bc.Outpoint
	bc.AssetAmount
	ControlProgram []byte

	AccountID           string
	ControlProgramIndex uint64
}

func (u *utxo) source() source {
	return source{AssetID: u.AssetID, AccountID: u.AccountID}
}

// source describes the criteria to use when selecting UTXOs.
type source struct {
	AssetID   bc.AssetID
	AccountID string
}

// reservation describes a reservation of a set of UTXOs belonging
// to a particular account. Reservations are immutable.
type reservation struct {
	ID          uint64
	Source      source
	UTXOs       []*utxo
	Change      uint64
	Expiry      time.Time
	ClientToken *string
}

func newReserver(db pg.DB, c *protocol.Chain, pinStore *pin.Store) *reserver {
	return &reserver{
		c:            c,
		db:           db,
		pinStore:     pinStore,
		reservations: make(map[uint64]*reservation),
		sources:      make(map[source]*sourceReserver),
	}
}

// reserver implements a utxo reserver that stores reservations
// in-memory. It relies on the account_utxos table for the source of
// truth of valid UTXOs but tracks which of those UTXOs are reserved
// in-memory.
//
// To reduce latency and prevent deadlock, no two mutexes (either on
// reserver or sourceReserver) should be held at the same time
//
// reserver ensures idempotency of reservations until the reservation
// expiration.
type reserver struct {
	c                 *protocol.Chain
	db                pg.DB
	pinStore          *pin.Store
	nextReservationID uint64
	idempotency       idempotency.Group

	reservationsMu sync.Mutex
	reservations   map[uint64]*reservation

	sourcesMu sync.Mutex
	sources   map[source]*sourceReserver
}

// Reserve selects and reserves UTXOs according to the critera provided
// in source. The resulting reservation expires at exp.
func (re *reserver) Reserve(ctx context.Context, src source, amount uint64, clientToken *string, exp time.Time) (*reservation, error) {
	if clientToken == nil {
		return re.reserve(ctx, src, amount, clientToken, exp)
	}

	untypedRes, err := re.idempotency.Once(*clientToken, func() (interface{}, error) {
		return re.reserve(ctx, src, amount, clientToken, exp)
	})
	return untypedRes.(*reservation), err
}

func (re *reserver) reserve(ctx context.Context, src source, amount uint64, clientToken *string, exp time.Time) (res *reservation, err error) {
	sourceReserver := re.source(src)

	// Try to reserve the right amount.
	rid := atomic.AddUint64(&re.nextReservationID, 1)
	reserved, total, err := sourceReserver.reserve(ctx, rid, amount)
	if err != nil {
		return nil, err
	}

	res = &reservation{
		ID:          rid,
		Source:      src,
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

// ReserveUTXO reserves a specific utxo for spending. The resulting
// reservation expires at exp.
func (re *reserver) ReserveUTXO(ctx context.Context, out bc.Outpoint, clientToken *string, exp time.Time) (*reservation, error) {
	if clientToken == nil {
		return re.reserveUTXO(ctx, out, exp, nil)
	}

	untypedRes, err := re.idempotency.Once(*clientToken, func() (interface{}, error) {
		return re.reserveUTXO(ctx, out, exp, clientToken)
	})
	return untypedRes.(*reservation), err
}

func (re *reserver) reserveUTXO(ctx context.Context, out bc.Outpoint, exp time.Time, clientToken *string) (*reservation, error) {
	u, err := findSpecificUTXO(ctx, re.db, out)
	if err != nil {
		return nil, err
	}
	if !re.checkUTXO(u) {
		return nil, pg.ErrUserInputNotFound
	}

	rid := atomic.AddUint64(&re.nextReservationID, 1)
	err = re.source(u.source()).reserveUTXO(rid, u)
	if err != nil {
		return nil, err
	}

	res := &reservation{
		ID:          rid,
		Source:      u.source(),
		UTXOs:       []*utxo{u},
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
func (re *reserver) Cancel(ctx context.Context, rid uint64) error {
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
func (re *reserver) ExpireReservations(ctx context.Context) error {
	// Remove records of any reservations that have expired.
	now := time.Now()
	var canceled []*reservation
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

func (re *reserver) checkUTXO(u *utxo) bool {
	_, s := re.c.State()
	return s.Tree.ContainsKey(state.OutputKey(u.Outpoint))
}

func (re *reserver) source(src source) *sourceReserver {
	re.sourcesMu.Lock()
	defer re.sourcesMu.Unlock()

	sr, ok := re.sources[src]
	if ok {
		return sr
	}

	sr = &sourceReserver{
		db:      re.db,
		src:     src,
		validFn: re.checkUTXO,
		heightFn: func() uint64 {
			return re.pinStore.Height(PinName)
		},
		cached:   make(map[bc.Outpoint]*utxo),
		reserved: make(map[bc.Outpoint]uint64),
	}
	re.sources[src] = sr
	return sr
}

type sourceReserver struct {
	db       pg.DB
	src      source
	validFn  func(u *utxo) bool
	heightFn func() uint64

	mu         sync.Mutex
	cached     map[bc.Outpoint]*utxo
	reserved   map[bc.Outpoint]uint64
	lastHeight uint64
}

func (sr *sourceReserver) reserve(ctx context.Context, rid uint64, amount uint64) ([]*utxo, uint64, error) {
	reservedUTXOs, reservedAmount, err := sr.reserveFromCache(rid, amount)
	if err == nil {
		return reservedUTXOs, reservedAmount, nil
	}

	// Find the set of UTXOs that match this source.
	err = sr.refillCache(ctx)
	if err != nil {
		return nil, 0, err
	}

	return sr.reserveFromCache(rid, amount)
}

func (sr *sourceReserver) reserveFromCache(rid uint64, amount uint64) ([]*utxo, uint64, error) {
	var (
		reserved, unavailable uint64
		reservedUTXOs         []*utxo
	)
	sr.mu.Lock()
	defer sr.mu.Unlock()

	for o, u := range sr.cached {
		// If the UTXO is already reserved, skip it.
		if _, ok := sr.reserved[u.Outpoint]; ok {
			unavailable += u.Amount
			continue
		}
		// Cached utxos aren't guaranteed to still be valid; they may
		// have been spent. Verify that that the outputs are still in
		// the state tree.
		if !sr.validFn(u) {
			delete(sr.cached, o)
			continue
		}

		reserved += u.Amount
		reservedUTXOs = append(reservedUTXOs, u)
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
	for _, u := range reservedUTXOs {
		sr.reserved[u.Outpoint] = rid
	}

	return reservedUTXOs, reserved, nil
}

func (sr *sourceReserver) reserveUTXO(rid uint64, utxo *utxo) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	_, isReserved := sr.reserved[utxo.Outpoint]
	if isReserved {
		return ErrReserved
	}

	sr.reserved[utxo.Outpoint] = rid
	return nil
}

func (sr *sourceReserver) cancel(res *reservation) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	for _, utxo := range res.UTXOs {
		delete(sr.reserved, utxo.Outpoint)
	}
}

func (sr *sourceReserver) refillCache(ctx context.Context) error {
	sr.mu.Lock()
	lastHeight := sr.lastHeight
	sr.mu.Unlock()

	curHeight := sr.heightFn()
	if lastHeight >= curHeight {
		return nil
	}

	utxos, err := findMatchingUTXOs(ctx, sr.db, sr.src, lastHeight)
	if err != nil {
		return errors.Wrap(err)
	}

	sr.mu.Lock()
	if curHeight > sr.lastHeight {
		sr.lastHeight = curHeight
	}
	for _, u := range utxos {
		sr.cached[u.Outpoint] = u
	}
	sr.mu.Unlock()

	return nil
}

func findMatchingUTXOs(ctx context.Context, db pg.DB, src source, height uint64) ([]*utxo, error) {
	const q = `
		SELECT tx_hash, index, amount, control_program_index, control_program
		FROM account_utxos
		WHERE account_id = $1 AND asset_id = $2 AND confirmed_in > $3
	`
	var utxos []*utxo
	err := pg.ForQueryRows(ctx, db, q, src.AccountID, src.AssetID, height,
		func(txHash bc.Hash, index uint32, amount uint64, cpIndex uint64, controlProg []byte) {
			utxos = append(utxos, &utxo{
				Outpoint: bc.Outpoint{
					Hash:  txHash,
					Index: index,
				},
				AssetAmount: bc.AssetAmount{
					Amount:  amount,
					AssetID: src.AssetID,
				},
				ControlProgram:      controlProg,
				AccountID:           src.AccountID,
				ControlProgramIndex: cpIndex,
			})
		})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return utxos, nil
}

func findSpecificUTXO(ctx context.Context, db pg.DB, out bc.Outpoint) (*utxo, error) {
	const q = `
		SELECT account_id, asset_id, amount, control_program_index, control_program
		FROM account_utxos
		WHERE tx_hash = $1 AND index = $2
	`
	u := new(utxo)
	err := db.QueryRow(ctx, q, out.Hash, out.Index).Scan(&u.AccountID, &u.AssetID, &u.Amount, &u.ControlProgramIndex, &u.ControlProgram)
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	} else if err != nil {
		return nil, errors.Wrap(err)
	}
	u.Outpoint = out
	return u, nil
}
