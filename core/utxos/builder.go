package utxos

import (
	"context"
	"encoding/json"

	"chain/core/txbuilder"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
)

type spendUTXOAction struct {
	store    *Store
	OutputID *bc.Hash `json:"output_id"`

	ReferenceData chainjson.Map `json:"reference_data"`
}

func (s *Store) NewSpendUTXOAction(outputID *bc.Hash, refData chainjson.Map) txbuilder.Action {
	return &spendUTXOAction{
		store:         s,
		OutputID:      outputID,
		ReferenceData: refData,
	}
}

func (s *Store) DecodeSpendUTXOAction(data []byte) (txbuilder.Action, error) {
	a := &spendUTXOAction{store: s}
	err := json.Unmarshal(data, a)
	return a, err
}

func (a *spendUTXOAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	if a.OutputID == nil {
		return txbuilder.MissingFieldsError("output_id")
	}

	var (
		sourceID       bc.Hash
		assetID        bc.AssetID
		amount         uint64
		sourcePos      uint64
		controlProgram []byte
		refDataHash    bc.Hash
	)
	const q = `
		SELECT source_id, asset_id, amount, source_pos, control_program, ref_data_hash
		FROM utxos WHERE output_id=$1
	`
	err := a.store.DB.QueryRowContext(ctx, q, a.OutputID).Scan(&sourceID, &assetID, &amount, &sourcePos, &controlProgram, &refDataHash)
	if err != nil {
		return errors.Wrap(err)
	}

	txInput := legacy.NewSpendInput(nil, sourceID, assetID, amount, sourcePos, controlProgram, refDataHash, a.ReferenceData)

	return b.AddInput(txInput, &txbuilder.SigningInstruction{})
}
