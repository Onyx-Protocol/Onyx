package query

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vmutil"
)

func transactionObject(orig *bc.Tx, b *bc.Block, indexInBlock uint32) map[string]interface{} {
	m := map[string]interface{}{
		"id":             orig.Hash.String(),
		"timestamp":      b.Time().Format(time.RFC3339),
		"block_id":       b.Hash().String(),
		"block_height":   b.Height,
		"position":       indexInBlock,
		"reference_data": unmarshalReferenceData(orig.ReferenceData),
	}

	inputs := make([]interface{}, 0, len(orig.Inputs))
	for _, in := range orig.Inputs {
		inputs = append(inputs, transactionInput(in))
	}
	outputs := make([]interface{}, 0, len(orig.Outputs))
	for i, out := range orig.Outputs {
		outputs = append(outputs, transactionOutput(out, uint32(i)))
	}
	m["inputs"] = inputs
	m["outputs"] = outputs

	return m
}

func transactionInput(in *bc.TxInput) map[string]interface{} {
	obj := map[string]interface{}{
		"asset_id":       in.AssetID().String(),
		"amount":         in.Amount(),
		"reference_data": unmarshalReferenceData(in.ReferenceData),
		"input_witness":  hexSlices(in.Arguments()),
	}
	if in.IsIssuance() {
		obj["action"] = "issue"
		obj["issuance_program"] = hex.EncodeToString(in.IssuanceProgram())
	} else {
		outpoint := in.Outpoint()
		obj["action"] = "spend"
		obj["control_program"] = hex.EncodeToString(in.ControlProgram())
		obj["spent_output"] = map[string]interface{}{
			"transaction_id": outpoint.Hash.String(),
			"position":       outpoint.Index,
		}
	}
	return obj
}

func transactionOutput(out *bc.TxOutput, idx uint32) map[string]interface{} {
	obj := map[string]interface{}{
		"position":        idx,
		"asset_id":        out.AssetID.String(),
		"amount":          out.Amount,
		"control_program": hex.EncodeToString(out.ControlProgram),
		"reference_data":  unmarshalReferenceData(out.ReferenceData),
	}

	if vmutil.IsUnspendable(out.ControlProgram) {
		obj["action"] = "retire"
	} else {
		obj["action"] = "control"
	}
	return obj
}

func unmarshalReferenceData(data []byte) map[string]interface{} {
	var obj map[string]interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		// Fall back to empty object
		return map[string]interface{}{}
	}
	return obj
}

func hexSlices(byteas [][]byte) []interface{} {
	res := make([]interface{}, 0, len(byteas))
	for _, s := range byteas {
		res = append(res, hex.EncodeToString(s))
	}
	return res
}

// localAnnotator depends on the asset and account annotators and
// must be run after them.
func localAnnotator(ctx context.Context, txs []map[string]interface{}) error {
	for _, tx := range txs {
		txIsLocal := "no"

		ins, ok := tx["inputs"].([]interface{})
		if !ok {
			return errors.Wrap(fmt.Errorf("bad inputs type %T", tx["inputs"]))
		}
		for _, inObj := range ins {
			in, ok := inObj.(map[string]interface{})
			if !ok {
				return errors.Wrap(fmt.Errorf("bad input type %T", inObj))
			}
			action, ok := in["action"].(string)
			if !ok {
				return errors.Wrap(fmt.Errorf("bad input action %T", in["action"]))
			}
			assetIsLocal, ok := in["asset_is_local"].(string)
			if !ok {
				return errors.Wrap(fmt.Errorf("bad input asset_is_local field: %T", in["asset_is_local"]))
			}

			_, hasAccount := in["account_id"]
			if (action == "issue" && assetIsLocal == "yes") || hasAccount {
				txIsLocal = "yes"
				in["is_local"] = "yes"
			} else {
				in["is_local"] = "no"
			}
		}

		outs, ok := tx["outputs"].([]interface{})
		if !ok {
			return errors.Wrap(fmt.Errorf("bad outputs type %T", tx["outputs"]))
		}
		for _, outObj := range outs {
			out, ok := outObj.(map[string]interface{})
			if !ok {
				return errors.Wrap(fmt.Errorf("bad output type %T", outObj))
			}

			_, hasAccount := out["account_id"]
			if hasAccount {
				txIsLocal = "yes"
				out["is_local"] = "yes"
			} else {
				out["is_local"] = "no"
			}
		}

		tx["is_local"] = txIsLocal
	}
	return nil
}
