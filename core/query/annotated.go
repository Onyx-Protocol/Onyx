package query

import (
	"encoding/hex"
	"encoding/json"

	"chain/cos/bc"
	"chain/cos/txscript"
)

func transactionObject(orig *bc.Tx, b *bc.Block, indexInBlock uint32) map[string]interface{} {
	m := map[string]interface{}{
		"id":             orig.Hash.String(),
		"block_id":       b.Hash().String(),
		"block_height":   b.Height,
		"position":       indexInBlock,
		"reference_data": unmarshalReferenceData(orig.Metadata),
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
	}
	if in.IsIssuance() {
		issuance := in.InputCommitment.(*bc.IssuanceInputCommitment)
		obj["action"] = "issue"
		obj["issuance_program"] = hex.EncodeToString(issuance.IssuanceProgram)
	} else {
		spend := in.InputCommitment.(*bc.SpendInputCommitment)
		obj["action"] = "spend"
		obj["spent_output"] = map[string]interface{}{
			"transaction_id": spend.Outpoint.Hash.String(),
			"position":       spend.Outpoint.Index,
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

	if txscript.IsUnspendable(out.ControlProgram) {
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
