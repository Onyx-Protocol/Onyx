package api

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
)

// /v3/buckets/:bucketID/addresses
func createAddr(ctx context.Context, bucketID string, in struct {
	Amount  uint64
	Expires time.Time
}) (interface{}, error) {
	addr := &appdb.Address{
		BucketID: bucketID,
		Amount:   in.Amount,
		Expires:  in.Expires,
		IsChange: false,
	}
	err := asset.CreateAddress(ctx, addr)
	if err != nil {
		return nil, err
	}

	signers := asset.Signers(addr.Keys, asset.ReceiverPath(addr))
	ret := map[string]interface{}{
		"address":             addr.Address,
		"signatures_required": addr.SigsRequired,
		"signers":             addrSigners(signers),
		"block_chain":         "sandbox",
		"created":             addr.Created.UTC(),
		"expires":             optionalTime(addr.Expires),
		"id":                  addr.ID,
		"index":               addr.Index[:],
	}
	return ret, nil
}

func addrSigners(signers []*asset.DerivedKey) (v []interface{}) {
	for _, s := range signers {
		v = append(v, map[string]interface{}{
			"pubkey":          s.Address.String(),
			"derivation_path": s.Path,
			"xpub":            s.Root.String(),
		})
	}
	return v
}
