package appdb

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/metrics"
)

// Bucket represents an indexed namespace inside of a wallet
type Bucket struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Index []uint32 `json:"account_index"`
}

// CreateBucket inserts a bucket database record
// for the given wallet,
// and returns the new Bucket.
func CreateBucket(ctx context.Context, walletID, label string) (*Bucket, error) {
	defer metrics.RecordElapsed(time.Now())
	if label == "" {
		return nil, ErrBadLabel
	}

	bucket := &Bucket{Label: label}

	const attempts = 3
	for i := 0; i < attempts; i++ {
		const q = `
			WITH incr AS (
				UPDATE manager_nodes
				SET
					accounts_count=accounts_count+1,
					next_account_index=next_account_index+1
				WHERE id=$1
				RETURNING (next_account_index - 1)
			)
			INSERT INTO accounts (manager_node_id, key_index, label)
			VALUES ($1, (TABLE incr), $2)
			RETURNING id, key_index(key_index)
		`
		err := pg.FromContext(ctx).QueryRow(q, walletID, label).Scan(
			&bucket.ID,
			(*pg.Uint32s)(&bucket.Index),
		)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			// There was an (expected) unique index conflict.
			// It is safe to try again.
			// This happens when there is contention incrementing
			// the bucket index.
			log.Write(ctx, "attempt", i, "error", err)
			if i == attempts-1 {
				return nil, err
			}
			continue
		} else if err != nil {
			return nil, err
		}
		break
	}

	return bucket, nil
}

// BucketBalance fetches the balances of assets contained in this bucket.
// It returns a slice of Balances and the last asset ID in the page.
// Each Balance contains an asset ID, a confirmed balance,
// and a total balance. The total and confirmed balances
// are currently the same.
func BucketBalance(ctx context.Context, bucketID, prev string, limit int) ([]*Balance, string, error) {
	q := `
		SELECT asset_id, sum(amount)::bigint
		FROM utxos
		WHERE account_id=$1 AND ($2='' OR asset_id>$2)
		GROUP BY asset_id
		ORDER BY asset_id
		LIMIT $3
	`
	rows, err := pg.FromContext(ctx).Query(q, bucketID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "balance query")
	}
	defer rows.Close()
	var (
		bals []*Balance
		last string
	)

	for rows.Next() {
		var (
			assetID string
			bal     int64
		)
		err = rows.Scan(&assetID, &bal)
		if err != nil {
			return nil, "", errors.Wrap(err, "row scan")
		}
		bals = append(bals, &Balance{assetID, bal, bal})
		last = assetID
	}
	if err = rows.Err(); err != nil {
		return nil, "", errors.Wrap(err, "rows error")
	}
	return bals, last, err
}

// ListBuckets returns a list of buckets contained in the given wallet.
func ListBuckets(ctx context.Context, walletID string, prev string, limit int) ([]*Bucket, string, error) {
	q := `
		SELECT id, label, key_index(key_index)
		FROM accounts
		WHERE manager_node_id = $1 AND ($2='' OR id<$2)
		ORDER BY id DESC LIMIT $3
	`
	rows, err := pg.FromContext(ctx).Query(q, walletID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var (
		buckets []*Bucket
		last    string
	)
	for rows.Next() {
		b := new(Bucket)
		err = rows.Scan(
			&b.ID,
			&b.Label,
			(*pg.Uint32s)(&b.Index),
		)
		if err != nil {
			return nil, "", errors.Wrap(err, "row scan")
		}
		buckets = append(buckets, b)
		last = b.ID
	}

	if err = rows.Err(); err != nil {
		return nil, "", errors.Wrap(err, "end row scan loop")
	}

	return buckets, last, err
}

// GetBucket returns a single bucket.
func GetBucket(ctx context.Context, bucketID string) (*Bucket, error) {
	q := `
		SELECT label, key_index(key_index)
		FROM accounts
		WHERE id = $1
	`
	b := &Bucket{ID: bucketID}
	err := pg.FromContext(ctx).QueryRow(q, bucketID).Scan(&b.Label, (*pg.Uint32s)(&b.Index))
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}

	return b, nil
}

// UpdateAccount updates the label of an account.
func UpdateAccount(ctx context.Context, accountID string, label *string) error {
	if label == nil {
		return nil
	}
	const q = `UPDATE accounts SET label = $2 WHERE id = $1`
	db := pg.FromContext(ctx)
	_, err := db.Exec(q, accountID, *label)
	return errors.Wrap(err, "update query")
}
