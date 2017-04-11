package asset

import (
	"context"
	"encoding/json"

	"github.com/lib/pq"

	"chain/core/query"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol/bc"
)

func (reg *Registry) AnnotateTxs(ctx context.Context, txs []*query.AnnotatedTx) error {
	assetIDMap := make(map[bc.AssetID]bool)

	// Collect all of the asset IDs appearing in the entire block. We only
	// check the outputs because every transaction should balance.
	for _, tx := range txs {
		for _, out := range tx.Outputs {
			assetIDMap[out.AssetID] = true
		}
	}
	if len(assetIDMap) == 0 {
		return nil
	}

	// Look up all the asset tags for all applicable assets.
	assetIDs := make([][]byte, 0, len(assetIDMap))
	for assetID := range assetIDMap {
		aid := assetID
		assetIDs = append(assetIDs, aid.Bytes())
	}
	var (
		tagsByAssetID    = make(map[bc.AssetID]*json.RawMessage, len(assetIDs))
		defsByAssetID    = make(map[bc.AssetID]*json.RawMessage, len(assetIDs))
		aliasesByAssetID = make(map[bc.AssetID]string, len(assetIDs))
		localByAssetID   = make(map[bc.AssetID]bool, len(assetIDs))
	)
	const q = `
		SELECT id, COALESCE(alias, ''), signer_id IS NOT NULL, tags, definition
		FROM assets
		LEFT JOIN asset_tags ON asset_id=id
		WHERE id IN (SELECT unnest($1::bytea[]))
	`
	err := pg.ForQueryRows(ctx, reg.db, q, pq.ByteaArray(assetIDs),
		func(assetID bc.AssetID, alias string, local bool, tagsBlob, defBlob []byte) error {
			if alias != "" {
				aliasesByAssetID[assetID] = alias
			}
			localByAssetID[assetID] = local

			jsonTags := json.RawMessage(tagsBlob)
			jsonDef := json.RawMessage(defBlob)
			if len(tagsBlob) > 0 {
				var v interface{}
				err := json.Unmarshal(tagsBlob, &v)
				if err == nil {
					tagsByAssetID[assetID] = &jsonTags
				}
			}
			if len(defBlob) > 0 {
				var v interface{}
				err := json.Unmarshal(defBlob, &v)
				if err == nil {
					defsByAssetID[assetID] = &jsonDef
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
			if alias, ok := aliasesByAssetID[in.AssetID]; ok {
				in.AssetAlias = alias
			}
			if localByAssetID[in.AssetID] {
				in.AssetIsLocal = true
			}
			tags := tagsByAssetID[in.AssetID]
			def := defsByAssetID[in.AssetID]
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
			if alias, ok := aliasesByAssetID[out.AssetID]; ok {
				out.AssetAlias = alias
			}
			if localByAssetID[out.AssetID] {
				out.AssetIsLocal = true
			}
			tags := tagsByAssetID[out.AssetID]
			def := defsByAssetID[out.AssetID]
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
