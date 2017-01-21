package query

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"time"

	chainjson "chain/encoding/json"
	"chain/protocol/bc"
	"chain/protocol/vmutil"
)

type AnnotatedTx struct {
	ID            chainjson.HexBytes `json:"id"`
	Timestamp     time.Time          `json:"timestamp"`
	BlockID       chainjson.HexBytes `json:"block_id"`
	BlockHeight   uint64             `json:"block_height"`
	Position      uint32             `json:"position"`
	ReferenceData *json.RawMessage   `json:"reference_data"`
	IsLocal       Bool               `json:"is_local"`
	Inputs        []*AnnotatedInput  `json:"inputs"`
	Outputs       []*AnnotatedOutput `json:"outputs"`
}

type AnnotatedInput struct {
	Type            string             `json:"type"`
	AssetID         chainjson.HexBytes `json:"asset_id"`
	AssetAlias      string             `json:"asset_alias,omitempty"`
	AssetDefinition *json.RawMessage   `json:"asset_definition"`
	AssetTags       *json.RawMessage   `json:"asset_tags,omitempty"`
	AssetIsLocal    Bool               `json:"asset_is_local"`
	Amount          uint64             `json:"amount"`
	IssuanceProgram chainjson.HexBytes `json:"issuance_program,omitempty"`
	ControlProgram  chainjson.HexBytes `json:"control_program,omitempty"`
	SpentOutputID   chainjson.HexBytes `json:"spent_output_id,omitempty"`
	AccountID       string             `json:"account_id,omitempty"`
	AccountAlias    string             `json:"account_alias,omitempty"`
	AccountTags     *json.RawMessage   `json:"account_tags,omitempty"`
	ReferenceData   *json.RawMessage   `json:"reference_data"`
	IsLocal         Bool               `json:"is_local"`
}

type AnnotatedOutput struct {
	Type            string             `json:"type"`
	Purpose         string             `json:"purpose,omitempty"`
	OutputID        chainjson.HexBytes `json:"output_id"`
	TransactionID   chainjson.HexBytes `json:"transaction_id,omitempty"`
	Position        uint32             `json:"position"`
	AssetID         chainjson.HexBytes `json:"asset_id"`
	AssetAlias      string             `json:"asset_alias,omitempty"`
	AssetDefinition *json.RawMessage   `json:"asset_definition"`
	AssetTags       *json.RawMessage   `json:"asset_tags"`
	AssetIsLocal    Bool               `json:"asset_is_local"`
	Amount          uint64             `json:"amount"`
	AccountID       string             `json:"account_id,omitempty"`
	AccountAlias    string             `json:"account_alias,omitempty"`
	AccountTags     *json.RawMessage   `json:"account_tags,omitempty"`
	ControlProgram  chainjson.HexBytes `json:"control_program"`
	ReferenceData   *json.RawMessage   `json:"reference_data"`
	IsLocal         Bool               `json:"is_local"`
}

type Bool bool

func (b Bool) MarshalJSON() ([]byte, error) {
	if b {
		return []byte(`"yes"`), nil
	}
	return []byte(`"no"`), nil
}

func (b *Bool) UnmarshalJSON(raw []byte) error {
	*b = false
	if bytes.Equal(raw, []byte(`"yes"`)) {
		*b = true
	}
	return nil
}

func buildAnnotatedTransaction(orig *bc.Tx, b *bc.Block, indexInBlock uint32) *AnnotatedTx {
	blockHash := b.Hash()
	referenceData := json.RawMessage(orig.ReferenceData)
	if len(referenceData) == 0 {
		referenceData = []byte(`{}`)
	}

	tx := &AnnotatedTx{
		ID:            orig.Hash[:],
		Timestamp:     b.Time(),
		BlockID:       blockHash[:],
		BlockHeight:   b.Height,
		Position:      indexInBlock,
		ReferenceData: &referenceData,
		Inputs:        make([]*AnnotatedInput, 0, len(orig.Inputs)),
		Outputs:       make([]*AnnotatedOutput, 0, len(orig.Outputs)),
	}
	for _, in := range orig.Inputs {
		tx.Inputs = append(tx.Inputs, buildAnnotatedInput(in))
	}
	for i, out := range orig.Outputs {
		tx.Outputs = append(tx.Outputs, buildAnnotatedOutput(out, uint32(i), orig.Hash))
	}
	return tx
}

func buildAnnotatedInput(orig *bc.TxInput) *AnnotatedInput {
	aid := orig.AssetID()

	referenceData := json.RawMessage(orig.ReferenceData)
	if len(referenceData) == 0 {
		referenceData = []byte(`{}`)
	}
	in := &AnnotatedInput{
		AssetID:       aid[:],
		Amount:        orig.Amount(),
		ReferenceData: &referenceData,
	}

	if orig.IsIssuance() {
		prog := orig.IssuanceProgram()
		in.Type = "issue"
		in.IssuanceProgram = prog
	} else {
		prog := orig.ControlProgram()
		prevoutID := orig.SpentOutputID()
		in.Type = "spend"
		in.ControlProgram = prog
		in.SpentOutputID = prevoutID[:]
	}
	return in
}

func buildAnnotatedOutput(orig *bc.TxOutput, idx uint32, txhash bc.Hash) *AnnotatedOutput {
	referenceData := json.RawMessage(orig.ReferenceData)
	if len(referenceData) == 0 {
		referenceData = []byte(`{}`)
	}
	outid := bc.ComputeOutputID(txhash, idx)
	out := &AnnotatedOutput{
		OutputID:       outid[:],
		Position:       idx,
		AssetID:        orig.AssetID[:],
		Amount:         orig.Amount,
		ControlProgram: orig.ControlProgram,
		ReferenceData:  &referenceData,
	}
	if vmutil.IsUnspendable(out.ControlProgram) {
		out.Type = "retire"
	} else {
		out.Type = "control"
	}
	return out
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
func localAnnotator(ctx context.Context, txs []*AnnotatedTx) {
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.AccountID != "" {
				tx.IsLocal, in.IsLocal = true, true
			}
			if in.Type == "issue" && in.AssetIsLocal {
				tx.IsLocal, in.IsLocal = true, true
			}
		}

		for _, out := range tx.Outputs {
			if out.AccountID != "" {
				tx.IsLocal, out.IsLocal = true, true
			}
		}
	}
}
