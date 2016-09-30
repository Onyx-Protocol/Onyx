package account

import (
	"context"
	"time"

	"chain/core/account/utxodb"
	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
)

type SpendAction struct {
	bc.AssetAmount
	AccountID string        `json:"account_id"`
	TxHash    *bc.Hash      `json:"transaction_id"`
	TxOut     *uint32       `json:"position"`
	TTL       time.Duration `json:"reservation_ttl"`

	// These fields are only necessary for filtering
	// aliases on transaction build requests. A wrapper
	// function reads them to set the ID fields. They are
	// not used anywhere else in the code base.
	AccountAlias string `json:"account_alias"`
	AssetAlias   string `json:"asset_alias"`

	ReferenceData json.Map `json:"reference_data"`
	ClientToken   *string  `json:"client_token"`
}

func (a *SpendAction) Build(ctx context.Context) (*txbuilder.BuildResult, error) {
	ttl := a.TTL
	if ttl == 0 {
		ttl = time.Minute
	}
	maxTime := time.Now().Add(ttl)

	acct, err := findByID(ctx, a.AccountID)
	if err != nil {
		return nil, errors.Wrap(err, "get account info")
	}

	utxodbSource := utxodb.Source{
		AssetID:     a.AssetID,
		Amount:      a.Amount,
		AccountID:   a.AccountID,
		TxHash:      a.TxHash,
		OutputIndex: a.TxOut,
		ClientToken: a.ClientToken,
	}
	utxodbSources := []utxodb.Source{utxodbSource}
	reserved, change, err := utxodb.Reserve(ctx, utxodbSources, maxTime)
	if err != nil {
		return nil, errors.Wrap(err, "reserving utxos")
	}

	var (
		txins      []*bc.TxInput
		tplInsts   []*txbuilder.SigningInstruction
		changeOuts []*bc.TxOutput
	)

	for _, r := range reserved {
		txInput, sigInst, err := utxoToInputs(ctx, acct, r, a.ReferenceData)
		if err != nil {
			return nil, errors.Wrap(err, "creating inputs")
		}

		txins = append(txins, txInput)
		tplInsts = append(tplInsts, sigInst)
	}
	if len(change) > 0 {
		acp, err := CreateControlProgram(ctx, a.AccountID, true)
		if err != nil {
			return nil, errors.Wrap(err, "creating control program")
		}
		changeOuts = append(changeOuts, bc.NewTxOutput(a.AssetID, change[0].Amount, acp, nil))
	}

	return &txbuilder.BuildResult{
		Inputs:              txins,
		Outputs:             changeOuts,
		SigningInstructions: tplInsts,
		MaxTimeMS:           bc.Millis(maxTime),
	}, nil
}

type SpendUTXOAction struct {
	TxHash bc.Hash       `json:"transaction_id"`
	TxOut  uint32        `json:"position"`
	TTL    time.Duration `json:"reservation_ttl"`

	ReferenceData json.Map `json:"reference_data"`
	ClientToken   *string  `json:"client_token"`
}

func (a *SpendUTXOAction) Build(ctx context.Context) (*txbuilder.BuildResult, error) {
	ttl := a.TTL
	if ttl == 0 {
		ttl = time.Minute
	}
	maxTime := time.Now().Add(ttl)

	r, err := utxodb.ReserveUTXO(ctx, a.TxHash, a.TxOut, a.ClientToken, maxTime)
	if err != nil {
		return nil, err
	}

	acct, err := findByID(ctx, r.AccountID)
	if err != nil {
		return nil, err
	}

	txInput, sigInst, err := utxoToInputs(ctx, acct, r, a.ReferenceData)
	if err != nil {
		return nil, err
	}

	return &txbuilder.BuildResult{
		Inputs:              []*bc.TxInput{txInput},
		SigningInstructions: []*txbuilder.SigningInstruction{sigInst},
		MaxTimeMS:           bc.Millis(maxTime),
	}, nil
}

func utxoToInputs(ctx context.Context, account *signers.Signer, u *utxodb.UTXO, refData []byte) (
	*bc.TxInput,
	*txbuilder.SigningInstruction,
	error,
) {
	txInput := bc.NewSpendInput(u.Hash, u.Index, nil, u.AssetID, u.Amount, u.Script, refData)

	sigInst := &txbuilder.SigningInstruction{
		AssetAmount: u.AssetAmount,
	}

	path := signers.Path(account, signers.AccountKeySpace, u.ControlProgramIndex)
	keyIDs := txbuilder.KeyIDs(account.XPubs, path)

	sigInst.AddWitnessKeys(keyIDs, account.Quorum)

	return txInput, sigInst, nil
}

type ControlAction struct {
	bc.AssetAmount
	AccountID string `json:"account_id"`

	// These fields are only necessary for filtering
	// aliases on transaction build requests. A wrapper
	// function reads them to set the ID fields. They are
	// not used anywhere else in the code base.
	AccountAlias string `json:"account_alias"`
	AssetAlias   string `json:"asset_alias"`

	ReferenceData json.Map `json:"reference_data"`
}

func (a *ControlAction) Build(ctx context.Context) (*txbuilder.BuildResult, error) {
	acp, err := CreateControlProgram(ctx, a.AccountID, false)
	if err != nil {
		return nil, err
	}
	out := bc.NewTxOutput(a.AssetID, a.Amount, acp, a.ReferenceData)
	return &txbuilder.BuildResult{Outputs: []*bc.TxOutput{out}}, nil
}

// CancelReservations cancels any existing reservations
// for the given outpoints.
func CancelReservations(ctx context.Context, outpoints []bc.Outpoint) error {
	return utxodb.Cancel(ctx, outpoints)
}
