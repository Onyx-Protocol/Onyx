package appdb

import (
	"database/sql"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/metrics"
)

// Address represents a blockchain address that is
// contained in an account.
type Address struct {
	// Initialized by Insert
	// (Insert reads all other fields)
	ID      string
	Created time.Time

	// Initialized by the package client
	Address      string // base58-encoded
	RedeemScript []byte
	PKScript     []byte
	Amount       uint64
	Expires      time.Time
	AccountID    string // read by LoadNextIndex
	IsChange     bool

	// Initialized by LoadNextIndex
	ManagerNodeID    string
	ManagerNodeIndex []uint32
	AccountIndex     []uint32
	Index            []uint32
	SigsRequired     int
	Keys             []*hdkey.XKey
}

var (
	// Map account ID to address template.
	// Entries set the following fields:
	//   ManagerNodeID
	//   ManagerNodeIndex
	//   AccountIndex
	//   Keys
	//   SigsRequired
	addrInfo      = map[string]*Address{}
	addrIndexNext int64 // next addr index in our block
	addrIndexCap  int64 // points to end of block
	addrMu        sync.Mutex
)

// AddrInfo looks up the information common to
// every address in the given account.
// Sets the following fields:
//   ManagerNodeID
//   ManagerNodeIndex
//   AccountIndex
//   Keys (comprised of both manager node keys and account keys)
//   SigsRequired
func AddrInfo(ctx context.Context, accountID string) (*Address, error) {
	addrMu.Lock()
	ai, ok := addrInfo[accountID]
	addrMu.Unlock()
	if !ok {
		// Concurrent cache misses might be doing
		// duplicate loads here, but ok.
		ai = new(Address)
		var nodeXPubs []string
		var accXPubs []string
		const q = `
		SELECT
			mn.id, key_index(a.key_index), key_index(mn.key_index),
			r.keyset, a.keys, mn.sigs_required
		FROM accounts a
		LEFT JOIN manager_nodes mn ON mn.id=a.manager_node_id
		LEFT JOIN rotations r ON r.id=mn.current_rotation
		WHERE a.id=$1
	`
		err := pg.FromContext(ctx).QueryRow(q, accountID).Scan(
			&ai.ManagerNodeID,
			(*pg.Uint32s)(&ai.AccountIndex),
			(*pg.Uint32s)(&ai.ManagerNodeIndex),
			(*pg.Strings)(&nodeXPubs),
			(*pg.Strings)(&accXPubs),
			&ai.SigsRequired,
		)
		if err != nil {
			return nil, errors.WithDetailf(err, "account %s", accountID)
		}

		ai.Keys, err = stringsToKeys(nodeXPubs)
		if err != nil {
			return nil, errors.Wrap(err, "parsing node keys")
		}

		if len(accXPubs) > 0 {
			accKeys, err := stringsToKeys(accXPubs)
			if err != nil {
				return nil, errors.Wrap(err, "parsing accountkeys")
			}

			ai.Keys = append(ai.Keys, accKeys...)
		}

		addrMu.Lock()
		addrInfo[accountID] = ai
		addrMu.Unlock()
	}
	return ai, nil
}

func nextIndex(ctx context.Context) ([]uint32, error) {
	defer metrics.RecordElapsed(time.Now())
	addrMu.Lock()
	defer addrMu.Unlock()

	if addrIndexNext >= addrIndexCap {
		var cap int64
		const incrby = 10000 // address_index_seq increments by 10,000
		const q = `SELECT nextval('address_index_seq')`
		err := pg.FromContext(ctx).QueryRow(q).Scan(&cap)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		addrIndexCap = cap
		addrIndexNext = cap - incrby
	}

	n := addrIndexNext
	addrIndexNext++
	return keyIndex(n), nil
}

func keyIndex(n int64) []uint32 {
	index := make([]uint32, 2)
	index[0] = uint32(n >> 31)
	index[1] = uint32(n & 0x7fffffff)
	return index
}

// LoadNextIndex is a low-level function to initialize a new Address.
// It is intended to be used by the asset package.
// Field AccountID must be set.
// LoadNextIndex will initialize some other fields;
// See Address for which ones.
func (a *Address) LoadNextIndex(ctx context.Context) error {
	defer metrics.RecordElapsed(time.Now())

	ai, err := AddrInfo(ctx, a.AccountID)
	if errors.Root(err) == sql.ErrNoRows {
		err = errors.Wrap(pg.ErrUserInputNotFound, err.Error())
	}
	if err != nil {
		return err
	}

	a.Index, err = nextIndex(ctx)
	if err != nil {
		return errors.Wrap(err, "nextIndex")
	}

	a.ManagerNodeID = ai.ManagerNodeID
	a.AccountIndex = ai.AccountIndex
	a.ManagerNodeIndex = ai.ManagerNodeIndex
	a.SigsRequired = ai.SigsRequired
	a.Keys = ai.Keys
	return nil
}

// Insert is a low-level function to insert an Address record.
// It is intended to be used by the asset package.
// Insert will initialize fields ID and Created;
// all other fields must be set prior to calling Insert.
func (a *Address) Insert(ctx context.Context) error {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		INSERT INTO addresses (
			address, redeem_script, pk_script, manager_node_id, account_id,
			keyset, expiration, amount, key_index, is_change
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, to_key_index($9), $10)
		RETURNING id, created_at;
	`
	row := pg.FromContext(ctx).QueryRow(q,
		a.Address,
		a.RedeemScript,
		a.PKScript,
		a.ManagerNodeID,
		a.AccountID,
		pg.Strings(keysToStrings(a.Keys)),
		pq.NullTime{Time: a.Expires, Valid: !a.Expires.IsZero()},
		a.Amount,
		pg.Uint32s(a.Index),
		a.IsChange,
	)
	return row.Scan(&a.ID, &a.Created)
}

func DeriveAddress(ctx context.Context, accountID string, addrIndex []uint32) (string, error) {
	addrInfo, err := AddrInfo(ctx, accountID)
	if err != nil {
		return "", errors.Wrap(err, "get addr info")
	}
	addr, _, err := hdkey.Address(addrInfo.Keys, ReceiverPath(addrInfo, addrIndex), addrInfo.SigsRequired)
	if err != nil {
		return "", errors.Wrap(err, "compute address")
	}
	return addr.String(), nil
}
