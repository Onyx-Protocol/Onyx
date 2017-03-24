package query

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"chain/crypto/ed25519/chainkd"
	chainjson "chain/encoding/json"
	"chain/protocol/bc"
	"chain/protocol/vmutil"
)

type AnnotatedTx struct {
	ID            bc.Hash            `json:"id"`
	Timestamp     time.Time          `json:"timestamp"`
	BlockID       bc.Hash            `json:"block_id"`
	BlockHeight   uint64             `json:"block_height"`
	Position      uint32             `json:"position"`
	ReferenceData *json.RawMessage   `json:"reference_data"`
	IsLocal       Bool               `json:"is_local"`
	Inputs        []*AnnotatedInput  `json:"inputs"`
	Outputs       []*AnnotatedOutput `json:"outputs"`
}

type AnnotatedInput struct {
	Type            string             `json:"type"`
	AssetID         bc.AssetID         `json:"asset_id"`
	AssetAlias      string             `json:"asset_alias,omitempty"`
	AssetDefinition *json.RawMessage   `json:"asset_definition"`
	AssetTags       *json.RawMessage   `json:"asset_tags,omitempty"`
	AssetIsLocal    Bool               `json:"asset_is_local"`
	Amount          uint64             `json:"amount"`
	IssuanceProgram chainjson.HexBytes `json:"issuance_program,omitempty"`
	ControlProgram  chainjson.HexBytes `json:"-"`
	SpentOutputID   *bc.Hash           `json:"spent_output_id,omitempty"`
	AccountID       string             `json:"account_id,omitempty"`
	AccountAlias    string             `json:"account_alias,omitempty"`
	AccountTags     *json.RawMessage   `json:"account_tags,omitempty"`
	ReferenceData   *json.RawMessage   `json:"reference_data"`
	IsLocal         Bool               `json:"is_local"`
}

type AnnotatedOutput struct {
	Type            string             `json:"type"`
	Purpose         string             `json:"purpose,omitempty"`
	OutputID        bc.Hash            `json:"id"`
	TransactionID   *bc.Hash           `json:"transaction_id,omitempty"`
	Position        uint32             `json:"position"`
	AssetID         bc.AssetID         `json:"asset_id"`
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

type AnnotatedAccount struct {
	ID     string           `json:"id"`
	Alias  string           `json:"alias,omitempty"`
	Keys   []*AccountKey    `json:"keys"`
	Quorum int              `json:"quorum"`
	Tags   *json.RawMessage `json:"tags"`
}

type AccountKey struct {
	RootXPub              chainkd.XPub         `json:"root_xpub"`
	AccountXPub           chainkd.XPub         `json:"account_xpub"`
	AccountDerivationPath []chainjson.HexBytes `json:"account_derivation_path"`
}

type AnnotatedAsset struct {
	ID              bc.AssetID         `json:"id"`
	Alias           string             `json:"alias,omitempty"`
	IssuanceProgram chainjson.HexBytes `json:"issuance_program"`
	Keys            []*AssetKey        `json:"keys"`
	Quorum          int                `json:"quorum"`
	Definition      *json.RawMessage   `json:"definition"`
	Tags            *json.RawMessage   `json:"tags"`
	IsLocal         Bool               `json:"is_local"`
}

type AssetKey struct {
	RootXPub            chainkd.XPub         `json:"root_xpub"`
	AssetPubkey         chainjson.HexBytes   `json:"asset_pubkey"`
	AssetDerivationPath []chainjson.HexBytes `json:"asset_derivation_path"`
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

var emptyJSONObject = json.RawMessage(`{}`)

func buildAnnotatedTransaction(orig *bc.Tx, b *bc.Block, indexInBlock uint32) *AnnotatedTx {
	tx := &AnnotatedTx{
		ID:            orig.ID,
		Timestamp:     b.Time(),
		BlockID:       b.Hash(),
		BlockHeight:   b.Height,
		Position:      indexInBlock,
		ReferenceData: &emptyJSONObject,
		Inputs:        make([]*AnnotatedInput, 0, len(orig.Inputs)),
		Outputs:       make([]*AnnotatedOutput, 0, len(orig.Outputs)),
	}
	if len(orig.ReferenceData) > 0 {
		referenceData := json.RawMessage(orig.ReferenceData)
		tx.ReferenceData = &referenceData
	}

	for i := range orig.Inputs {
		tx.Inputs = append(tx.Inputs, buildAnnotatedInput(orig, uint32(i)))
	}
	for i := range orig.Outputs {
		tx.Outputs = append(tx.Outputs, buildAnnotatedOutput(orig, uint32(i)))
	}
	return tx
}

func buildAnnotatedInput(tx *bc.Tx, i uint32) *AnnotatedInput {
	orig := tx.Inputs[i]
	in := &AnnotatedInput{
		AssetID:         orig.AssetID(),
		Amount:          orig.Amount(),
		AssetDefinition: &emptyJSONObject,
		AssetTags:       &emptyJSONObject,
		ReferenceData:   &emptyJSONObject,
	}

	if len(orig.ReferenceData) > 0 {
		referenceData := json.RawMessage(orig.ReferenceData)
		in.ReferenceData = &referenceData
	}

	if sp, ok := tx.TxEntries.TxInputs[i].(*bc.Spend); ok {
		in.Type = "spend"
		in.ControlProgram = orig.ControlProgram()
		in.SpentOutputID = &sp.Body.SpentOutputID
	} else {
		in.Type = "issue"
		in.IssuanceProgram = orig.IssuanceProgram()
	}
	return in
}

func buildAnnotatedOutput(tx *bc.Tx, idx uint32) *AnnotatedOutput {
	orig := tx.Outputs[idx]
	outid := tx.OutputID(idx)
	out := &AnnotatedOutput{
		OutputID:        outid,
		Position:        idx,
		AssetID:         orig.AssetID,
		AssetDefinition: &emptyJSONObject,
		AssetTags:       &emptyJSONObject,
		Amount:          orig.Amount,
		ControlProgram:  orig.ControlProgram,
		ReferenceData:   &emptyJSONObject,
	}
	if len(orig.ReferenceData) > 0 {
		referenceData := json.RawMessage(orig.ReferenceData)
		out.ReferenceData = &referenceData
	}
	if vmutil.IsUnspendable(out.ControlProgram) {
		out.Type = "retire"
	} else {
		out.Type = "control"
	}
	return out
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
