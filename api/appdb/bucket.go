package appdb

import (
	"github.com/lib/pq"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/log"
)

// Bucket represents an indexed namespace inside of a wallet
type Bucket struct {
	ID      string   `json:"bucket_id"`
	Label   string   `json:"label"`
	Index   []uint32 `json:"bucket_index"`
	Balance int64    `json:"balance"`
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
