package asset

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
)

func (reg *Registry) AnnotateTxs(ctx context.Context, txs []map[string]interface{}) error {
	assetIDStrMap := make(map[string]bool)

	// Collect all of the asset IDs appearing in the entire block. We only
	// check the outputs because every transaction should balance.
	for _, tx := range txs {
		outs, ok := tx["outputs"].([]interface{})
		if !ok {
			log.Error(ctx, errors.Wrap(fmt.Errorf("bad outputs type %T", tx["outputs"])))
			continue
		}
		for _, outObj := range outs {
			out, ok := outObj.(map[string]interface{})
			if !ok {
				log.Error(ctx, errors.Wrap(fmt.Errorf("bad output type %T", outObj)))
				continue
			}
			assetIDStr, ok := out["asset_id"].(string)
			if !ok {
				log.Error(ctx, errors.Wrap(fmt.Errorf("bad asset_id type %T", out["asset_id"])))
				continue
			}
			assetIDStrMap[assetIDStr] = true
		}
	}
	if len(assetIDStrMap) == 0 {
		return nil
	}

	// Look up all the asset tags for all applicable assets.
	assetIDStrs := make([]string, 0, len(assetIDStrMap))
	for assetIDStr := range assetIDStrMap {
		assetIDStrs = append(assetIDStrs, assetIDStr)
	}
	var (
		tagsByAssetIDStr    = make(map[string]map[string]interface{}, len(assetIDStrs))
		defsByAssetIDStr    = make(map[string]map[string]interface{}, len(assetIDStrs))
		aliasesByAssetIDStr = make(map[string]string, len(assetIDStrs))
		localByAssetIDStr   = make(map[string]bool, len(assetIDStrs))
	)
	const q = `
		SELECT encode(id, 'hex'), COALESCE(alias, ''), signer_id IS NOT NULL, tags, definition
		FROM assets
		LEFT JOIN asset_tags ON asset_id=id
		WHERE id IN (SELECT decode(unnest($1::text[]), 'hex'))
	`
	err := pg.ForQueryRows(ctx, reg.db, q, pq.StringArray(assetIDStrs),
		func(assetIDStr, alias string, local bool, tagsBlob []byte, defBlob []byte) error {
			if alias != "" {
				aliasesByAssetIDStr[assetIDStr] = alias
			}
			localByAssetIDStr[assetIDStr] = local
			if len(tagsBlob) > 0 {
				var tags map[string]interface{}
				err := json.Unmarshal(tagsBlob, &tags)
				if err != nil {
					return err
				}
				tagsByAssetIDStr[assetIDStr] = tags
			}
			if len(defBlob) > 0 {
				var def map[string]interface{}
				err := json.Unmarshal(defBlob, &def)
				if err == nil { // ignore non-json defs
					defsByAssetIDStr[assetIDStr] = def
				}
			}
			return nil
		},
	)
	if err != nil {
		return errors.Wrap(err, "querying assets")
	}

	empty := map[string]interface{}{}
	applyAnnotations := func(s interface{}) {
		asSlice, ok := s.([]interface{})
		if !ok {
			log.Error(ctx, errors.Wrap(fmt.Errorf("expectd slice, got %T", s)))
			return
		}
		for _, m := range asSlice {
			asMap, ok := m.(map[string]interface{})
			if !ok {
				log.Error(ctx, errors.Wrap(fmt.Errorf("bad input type %T", m)))
				continue
			}
			assetIDStr, ok := asMap["asset_id"].(string)
			if !ok {
				log.Error(ctx, errors.Wrap(fmt.Errorf("bad asset_id type %T", asMap["asset_id"])))
				continue
			}
			tags := tagsByAssetIDStr[assetIDStr]
			if tags != nil {
				asMap["asset_tags"] = tags
			} else {
				asMap["asset_tags"] = empty
			}
			if alias, ok := aliasesByAssetIDStr[assetIDStr]; ok {
				asMap["asset_alias"] = alias
			}
			if localByAssetIDStr[assetIDStr] {
				asMap["asset_is_local"] = "yes"
			} else {
				asMap["asset_is_local"] = "no"
			}
			def := defsByAssetIDStr[assetIDStr]
			if def != nil {
				asMap["asset_definition"] = def
			} else {
				asMap["asset_definition"] = empty
			}
		}
	}

	// Add the asset tags to all the inputs & outputs.
	for _, tx := range txs {
		applyAnnotations(tx["inputs"])
		applyAnnotations(tx["outputs"])
	}
	return nil
}
