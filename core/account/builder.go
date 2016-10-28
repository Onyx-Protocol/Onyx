package account

import (
	"context"
	"encoding/json"
	"time"

	"chain/core/account/utxodb"
	"chain/core/signers"
	"chain/core/txbuilder"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/log"
	"chain/protocol/bc"
)

func (m *Manager) NewSpendAction(amt bc.AssetAmount, accountID string, txHash *bc.Hash, txOut *uint32, refData chainjson.Map, clientToken *string) txbuilder.Action {
	return &spendAction{
		accounts:      m,
		AssetAmount:   amt,
		TxHash:        txHash,
		TxOut:         txOut,
		AccountID:     accountID,
		ReferenceData: refData,
		ClientToken:   clientToken,
	}
}

func (m *Manager) DecodeSpendAction(data []byte) (txbuilder.Action, error) {
	a := &spendAction{accounts: m}
	err := json.Unmarshal(data, a)
	return a, err
}

type spendAction struct {
	accounts *Manager
	bc.AssetAmount
	AccountID     string        `json:"account_id"`
	TxHash        *bc.Hash      `json:"transaction_id"`
	TxOut         *uint32       `json:"position"`
	ReferenceData chainjson.Map `json:"reference_data"`
	ClientToken   *string       `json:"client_token"`
}

func (a *spendAction) Build(ctx context.Context, maxTime time.Time) (*txbuilder.BuildResult, error) {
	acct, err := a.accounts.findByID(ctx, a.AccountID)
	if err != nil {
		return nil, errors.Wrap(err, "get account info")
	}

	src := utxodb.Source{
		AssetID:     a.AssetID,
		Amount:      a.Amount,
		AccountID:   a.AccountID,
		TxHash:      a.TxHash,
		OutputIndex: a.TxOut,
		ClientToken: a.ClientToken,
	}
	rid, reserved, change, err := a.accounts.utxoDB.Reserve(ctx, src, maxTime)
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
		acp, err := a.accounts.CreateControlProgram(ctx, a.AccountID, true)
		if err != nil {
			return nil, errors.Wrap(err, "creating control program")
		}
		changeOuts = append(changeOuts, bc.NewTxOutput(a.AssetID, change[0].Amount, acp, nil))
	}

	br := &txbuilder.BuildResult{
		Inputs:              txins,
		Outputs:             changeOuts,
		SigningInstructions: tplInsts,
		Rollback:            canceler(ctx, a.accounts, rid),
	}
	return br, nil
}

func (m *Manager) NewSpendUTXOAction(outpoint bc.Outpoint) txbuilder.Action {
	return &spendUTXOAction{
		accounts: m,
		TxHash:   outpoint.Hash,
		TxOut:    outpoint.Index,
	}
}

func (m *Manager) DecodeSpendUTXOAction(data []byte) (txbuilder.Action, error) {
	a := &spendUTXOAction{accounts: m}
	err := json.Unmarshal(data, a)
	return a, err
}

type spendUTXOAction struct {
	accounts *Manager
	TxHash   bc.Hash `json:"transaction_id"`
	TxOut    uint32  `json:"position"`

	ReferenceData chainjson.Map `json:"reference_data"`
	ClientToken   *string       `json:"client_token"`
}

func (a *spendUTXOAction) Build(ctx context.Context, maxTime time.Time) (*txbuilder.BuildResult, error) {
	rid, r, err := a.accounts.utxoDB.ReserveUTXO(ctx, a.TxHash, a.TxOut, a.ClientToken, maxTime)
	if err != nil {
		return nil, err
	}

	acct, err := a.accounts.findByID(ctx, r.AccountID)
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
		Rollback:            canceler(ctx, a.accounts, rid),
	}, nil
}

// Best-effort cancellation attempt to put in txbuilder.BuildResult.Rollback.
func canceler(ctx context.Context, m *Manager, rid int32) func() {
	return func() {
		err := m.utxoDB.Cancel(ctx, rid)
		if err != nil {
			log.Error(ctx, err)
		}
	}
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

func (m *Manager) NewControlAction(amt bc.AssetAmount, accountID string, refData chainjson.Map) txbuilder.Action {
	return &controlAction{
		accounts:      m,
		AssetAmount:   amt,
		AccountID:     accountID,
		ReferenceData: refData,
	}
}

func (m *Manager) DecodeControlAction(data []byte) (txbuilder.Action, error) {
	a := &controlAction{accounts: m}
	err := json.Unmarshal(data, a)
	return a, err
}

type controlAction struct {
	accounts *Manager
	bc.AssetAmount
	AccountID     string        `json:"account_id"`
	ReferenceData chainjson.Map `json:"reference_data"`
}

func (a *controlAction) Build(ctx context.Context, maxTime time.Time) (*txbuilder.BuildResult, error) {
	acp, err := a.accounts.CreateControlProgram(ctx, a.AccountID, false)
	if err != nil {
		return nil, err
	}
	out := bc.NewTxOutput(a.AssetID, a.Amount, acp, a.ReferenceData)
	return &txbuilder.BuildResult{Outputs: []*bc.TxOutput{out}}, nil
}
