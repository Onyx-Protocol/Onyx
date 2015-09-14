package appdb

import (
	"github.com/lib/pq"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
)

// Bucket represents an indexed namespace inside of a wallet
type Bucket struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Index []uint32 `json:"bucket_index"`
}

// CreateBucket inserts a bucket database record
// for the given wallet,
// and returns the new Bucket.
func CreateBucket(ctx context.Context, walletID, label string) (*Bucket, error) {
	if label == "" {
		return nil, ErrBadLabel
	}

	bucket := &Bucket{Label: label}

	const attempts = 3
	for i := 0; i < attempts; i++ {
		const q = `
			WITH incr AS (
				UPDATE wallets
				SET
					buckets_count=buckets_count+1,
					next_bucket_index=next_bucket_index+1
				WHERE id=$1
				RETURNING (next_bucket_index - 1)
			)
			INSERT INTO buckets (wallet_id, key_index, label)
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
// It returns a slice of Balances, where each Balance contains an asset ID,
// a confirmed balance, and a total balance. The total and confirmed balances
// are currently the same.
func BucketBalance(ctx context.Context, bucketID string) ([]*Balance, error) {
	q := `
		SELECT asset_id, sum(amount)::bigint
		FROM utxos
		WHERE bucket_id=$1
		GROUP BY asset_id
		ORDER BY asset_id
	`
	rows, err := pg.FromContext(ctx).Query(q, bucketID)
	if err != nil {
		return nil, errors.Wrap(err, "balance query")
	}
	defer rows.Close()
	var bals []*Balance

	for rows.Next() {
		var (
			assetID string
			bal     int64
		)
		err = rows.Scan(&assetID, &bal)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		b := &Balance{
			AssetID:   assetID,
			Total:     bal,
			Confirmed: bal,
		}
		bals = append(bals, b)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows error")
	}
	return bals, err
}

// ListBuckets returns a list of buckets contained in the given wallet.
func ListBuckets(ctx context.Context, walletID string) ([]*Bucket, error) {
	q := `
		SELECT id, label, key_index(key_index)
		FROM buckets
		WHERE wallet_id = $1
		ORDER BY created_at
	`
	rows, err := pg.FromContext(ctx).Query(q, walletID)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var buckets []*Bucket
	for rows.Next() {
		b := new(Bucket)
		err = rows.Scan(
			&b.ID,
			&b.Label,
			(*pg.Uint32s)(&b.Index),
		)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		buckets = append(buckets, b)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return buckets, err
}
