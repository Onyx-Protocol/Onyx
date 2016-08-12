package asset

import (
	"encoding/json"
	"fmt"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
)

func AnnotateTxs(ctx context.Context, txs []map[string]interface{}) error {
	assetIDStrMap := make(map[string]bool)

	for _, tx := range txs {
		// TODO(bobg): annotate inputs too

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
	assetIDStrs := make([]string, 0, len(assetIDStrMap))
	for assetIDStr, _ := range assetIDStrMap {
		assetIDStrs = append(assetIDStrs, assetIDStr)
	}
	tagsByAssetIDStr := make(map[string]map[string]interface{}, len(assetIDStrs))
	err := pg.ForQueryRows(ctx, `SELECT asset_id, tags FROM asset_tags WHERE asset_id IN (SELECT unnest($1::text[]))`, pg.Strings(assetIDStrs),
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
	for _, tx := range txs {
		outs := tx["outputs"].([]interface{}) // error check happened above
		for _, outObj := range outs {
			out := outObj.(map[string]interface{}) // error check happened above
			assetIDStr := out["asset_id"].(string) // error check happened above
			tags := tagsByAssetIDStr[assetIDStr]
			if tags != nil {
				out["asset_tags"] = tags
			}
		}
	}
	return nil
}
