package api

import (
	"io"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
)

// /v3/buckets/:bucketID/addresses
func createAddr(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var input struct {
		Amount  uint64
		Expires time.Time
	}

	bucketID := req.URL.Query().Get(":bucketID")
	err := readJSON(req.Body, &input)
	if err != nil && err != io.EOF {
		writeHTTPError(ctx, w, err)
		return
	}

	addr := &appdb.Address{
		BucketID: bucketID,
		Amount:   input.Amount,
		Expires:  input.Expires,
		IsChange: false,
	}
	err = asset.CreateAddress(ctx, addr)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	signers := asset.Signers(addr.Keys, asset.ReceiverPath(addr))
	writeJSON(ctx, w, 201, map[string]interface{}{
		"address":             addr.Address,
		"signatures_required": addr.SigsRequired,
		"signers":             addrSigners(signers),
		"block_chain":         "sandbox",
		"created":             addr.Created.UTC(),
		"expires":             optionalTime(addr.Expires),
		"receiver_id":         addr.ID,
		"index":               addr.Index[:],
	})
}

func addrSigners(signers []*asset.DerivedKey) (v []interface{}) {
	for _, s := range signers {
		v = append(v, map[string]interface{}{
			"pubkey":          s.Address.String(),
			"derivation_path": s.Path,
			"xpub_hash":       s.Root.ID,
		})
	}
	return v
}
