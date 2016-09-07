package asset

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/errors"
)

func AnnotateTxs(ctx context.Context, txs []map[string]interface{}) error {
	assetIDStrMap := make(map[string]bool)

	// Collect all of the asset IDs appearing in the entire block. We only
	// check the outputs because every transaction should balance.
	for _, tx := range txs {
		outs, ok := tx["outputs"].([]interface{})
		if !ok {
			return errors.Wrap(fmt.Errorf("bad outputs type %T", tx["outputs"]))
		}
		for _, outObj := range outs {
			out, ok := outObj.(map[string]interface{})
			if !ok {
				return errors.Wrap(fmt.Errorf("bad output type %T", outObj))
			}
			assetIDStr, ok := out["asset_id"].(string)
			if !ok {
				return errors.Wrap(fmt.Errorf("bad asset_id type %T", out["asset_id"]))
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
	tagsByAssetIDStr := make(map[string]map[string]interface{}, len(assetIDStrs))
	err := pg.ForQueryRows(ctx, `SELECT asset_id, tags FROM asset_tags WHERE asset_id IN (SELECT unnest($1::text[]))`, pq.StringArray(assetIDStrs),
		func(assetIDStr string, tagsBlob []byte) error {
			if len(tagsBlob) == 0 {
				return nil
			}
			var tags map[string]interface{}
			err := json.Unmarshal(tagsBlob, &tags)
			if err != nil {
				return err
			}
			tagsByAssetIDStr[assetIDStr] = tags
			return nil
		},
	)
	if err != nil {
		return errors.Wrap(err, "querying asset tags")
	}

	// Look up all the asset aliases for all applicable assets.
	aliasesByAssetIDStr := make(map[string]string, len(assetIDStrs))
	const aliasQ = `
		SELECT id, alias FROM assets
		WHERE alias IS NOT NULL AND id IN (SELECT unnest($1::text[]))
	`
	err = pg.ForQueryRows(ctx, aliasQ, pq.StringArray(assetIDStrs), func(assetIDStr, alias string) {
		aliasesByAssetIDStr[assetIDStr] = alias
	})
	if err != nil {
		return errors.Wrap(err, "querying asset aliases")
	}

	// Add the asset tags to all the inputs & outputs.
	empty := map[string]interface{}{}
	for _, tx := range txs {
		ins, ok := tx["inputs"].([]interface{})
		if !ok {
			return errors.Wrap(fmt.Errorf("bad inputs type %T", tx["inputs"]))
		}
		for _, inObj := range ins {
			in, ok := inObj.(map[string]interface{})
			if !ok {
				return errors.Wrap(fmt.Errorf("bad input type %T", inObj))
			}
			assetIDStr, ok := in["asset_id"].(string)
			if !ok {
				return errors.Wrap(fmt.Errorf("bad asset_id type %T", in["asset_id"]))
			}
			tags := tagsByAssetIDStr[assetIDStr]
			if tags != nil {
				in["asset_tags"] = tags
			} else {
				in["asset_tags"] = empty
			}
			if alias, ok := aliasesByAssetIDStr[assetIDStr]; ok {
				in["asset_alias"] = alias
			}
		}

		outs := tx["outputs"].([]interface{}) // error check happened above
		for _, outObj := range outs {
			out := outObj.(map[string]interface{}) // error check happened above
			assetIDStr := out["asset_id"].(string) // error check happened above
			tags := tagsByAssetIDStr[assetIDStr]
			if tags != nil {
				out["asset_tags"] = tags
			} else {
				out["asset_tags"] = empty
			}
			if alias, ok := aliasesByAssetIDStr[assetIDStr]; ok {
				out["asset_alias"] = alias
			}
		}
	}
	return nil
}
