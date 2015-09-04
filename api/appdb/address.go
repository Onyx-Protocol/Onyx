package appdb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/errors"
)

// Address represents a blockchain address that is
// contained in a bucket.
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
	BucketID     string // read by LoadNextIndex
	IsChange     bool

	// Initialized by LoadNextIndex
	WalletID     string
	WalletIndex  []uint32
	BucketIndex  []uint32
	Index        []uint32
	SigsRequired int
	Keys         []*Key
}

// LoadNextIndex is a low-level function to initialize a new Address.
// It is intended to be used by the asset package.
// Field BucketID must be set.
// LoadNextIndex will initialize some other fields;
// See Address for which ones.
func (a *Address) LoadNextIndex(ctx context.Context) error {
	var keyIDs []string
	const q = `
		SELECT
			w.id, key_index(b.key_index), key_index(w.key_index),
			r.keyset, w.sigs_required
		FROM buckets b
		LEFT JOIN wallets w ON w.id=b.wallet_id
		LEFT JOIN rotations r ON r.id=w.current_rotation
		WHERE b.id=$1
	`
	err := pg.FromContext(ctx).QueryRow(q, a.BucketID).Scan(
		&a.WalletID,
		(*pg.Uint32s)(&a.BucketIndex),
		(*pg.Uint32s)(&a.WalletIndex),
		(*pg.Strings)(&keyIDs),
		&a.SigsRequired,
	)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	if err != nil {
		return errors.WithDetailf(err, "bucket %s", a.BucketID)
	}
	if len(keyIDs) == 0 {
		// Postgres can't put a fk constraint on an array (eg keyset),
		// so we need to check this explicitly (using a LEFT JOIN above).
		return errors.New("could not load keys for bucket " + a.BucketID)
	}

	a.Keys, err = getKeys(ctx, keyIDs)
	if err != nil {
		return errors.Wrap(err, "get keys")
	}
	a.Index, err = newAddressIndex(ctx, a.BucketID)
	if err != nil {
		return errors.Wrap(err, "allocate index")
	}
	return nil
}

// Insert is a low-level function to insert an Address record.
// It is intended to be used by the asset package.
// Insert will initialize fields ID and Created;
// all other fields must be set prior to calling Insert.
func (a *Address) Insert(ctx context.Context) error {
	const q = `
		INSERT INTO addresses (
			address, redeem_script, pk_script, wallet_id, bucket_id,
			keyset, expiration, amount, key_index, is_change
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, to_key_index($9), $10)
		RETURNING id, created_at;
	`
	row := pg.FromContext(ctx).QueryRow(q,
		a.Address,
		a.RedeemScript,
		a.PKScript,
		a.WalletID,
		a.BucketID,
		pg.Strings(keyIDs(a.Keys)),
		pq.NullTime{Time: a.Expires, Valid: !a.Expires.IsZero()},
		a.Amount,
		pg.Uint32s(a.Index),
		a.IsChange,
	)
	return row.Scan(&a.ID, &a.Created)
}

// newAddressIndex allocates a new index for an address in bucket bID.
func newAddressIndex(ctx context.Context, bID string) (index []uint32, err error) {
	const q = `
		UPDATE buckets SET next_address_index = next_address_index + 1
		WHERE id = $1
		RETURNING key_index(next_address_index - 1)
	`
	err = pg.FromContext(ctx).QueryRow(q, bID).Scan((*pg.Uint32s)(&index))
	return
}
