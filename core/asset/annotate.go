package asset

import (
	"context"
	"encoding/json"

	"github.com/lib/pq"

	"chain/core/query"
	"chain/database/pg"
	"chain/errors"
)

func (reg *Registry) AnnotateTxs(ctx context.Context, txs []*query.AnnotatedTx) error {
	assetIDStrMap := make(map[string][]byte)

	// Collect all of the asset IDs appearing in the entire block. We only
	// check the outputs because every transaction should balance.
	for _, tx := range txs {
		for _, out := range tx.Outputs {
			assetIDStrMap[string(out.AssetID)] = out.AssetID
		}
	}
	if len(assetIDStrMap) == 0 {
		return nil
	}

	// Look up all the asset tags for all applicable assets.
	assetIDs := make([][]byte, 0, len(assetIDStrMap))
	for _, assetID := range assetIDStrMap {
		assetIDs = append(assetIDs, assetID)
	}
	var (
		tagsByAssetIDStr    = make(map[string]*json.RawMessage, len(assetIDs))
		defsByAssetIDStr    = make(map[string]*json.RawMessage, len(assetIDs))
		aliasesByAssetIDStr = make(map[string]string, len(assetIDs))
		localByAssetIDStr   = make(map[string]bool, len(assetIDs))
	)
	const q = `
		SELECT id, COALESCE(alias, ''), signer_id IS NOT NULL, tags, definition
		FROM assets
		LEFT JOIN asset_tags ON asset_id=id
		WHERE id IN (SELECT unnest($1::bytea[]))
	`
	err := pg.ForQueryRows(ctx, reg.db, q, pq.ByteaArray(assetIDs),
		func(assetID []byte, alias string, local bool, tagsBlob, defBlob []byte) error {
			assetIDStr := string(assetID)
			if alias != "" {
				aliasesByAssetIDStr[assetIDStr] = alias
			}
			localByAssetIDStr[assetIDStr] = local

			jsonTags := json.RawMessage(tagsBlob)
			jsonDef := json.RawMessage(defBlob)
			if len(tagsBlob) > 0 {
				var v interface{}
				err := json.Unmarshal(tagsBlob, &v)
				if err == nil {
					tagsByAssetIDStr[assetIDStr] = &jsonTags
				}
			}
			if len(defBlob) > 0 {
				var v interface{}
				err := json.Unmarshal(defBlob, &v)
				if err == nil {
					defsByAssetIDStr[assetIDStr] = &jsonDef
				}
			}
			return nil
		},
	)
	if err != nil {
		return errors.Wrap(err, "querying assets")
	}

	empty := json.RawMessage(`{}`)
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			assetIDStr := string(in.AssetID)

			if alias, ok := aliasesByAssetIDStr[assetIDStr]; ok {
				in.AssetAlias = alias
			}
			if localByAssetIDStr[assetIDStr] {
				in.AssetIsLocal = true
			}
			tags := tagsByAssetIDStr[assetIDStr]
			def := defsByAssetIDStr[assetIDStr]
			in.AssetTags = &empty
			in.AssetDefinition = &empty
			if tags != nil {
				in.AssetTags = tags
			}
			if def != nil {
				in.AssetDefinition = def
			}
		}

		for _, out := range tx.Outputs {
			assetIDStr := string(out.AssetID)

			if alias, ok := aliasesByAssetIDStr[assetIDStr]; ok {
				out.AssetAlias = alias
			}
			if localByAssetIDStr[assetIDStr] {
				out.AssetIsLocal = true
			}
			tags := tagsByAssetIDStr[assetIDStr]
			def := defsByAssetIDStr[assetIDStr]
			out.AssetTags = &empty
			out.AssetDefinition = &empty
			if tags != nil {
				out.AssetTags = tags
			}
			if def != nil {
				out.AssetDefinition = def
			}
		}
	}

	return nil
}
