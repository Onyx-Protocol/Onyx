package account

import (
	"context"
	"database/sql"
	"encoding/json"

	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/log"
	"chain/protocol/bc"
)

func (m *Manager) NewSpendAction(amt bc.AssetAmount, accountID string, refData chainjson.Map, clientToken *string) txbuilder.Action {
	return &spendAction{
		accounts:      m,
		AssetAmount:   amt,
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
	ReferenceData chainjson.Map `json:"reference_data"`
	ClientToken   *string       `json:"client_token"`
}

func (a *spendAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string
	if a.AccountID == "" {
		missing = append(missing, "account_id")
	}
	if a.AssetID == (bc.AssetID{}) {
		missing = append(missing, "asset_id")
	}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	acct, err := a.accounts.findByID(ctx, a.AccountID)
	if err != nil {
		return errors.Wrap(err, "get account info")
	}

	src := source{
		AssetID:   a.AssetID,
		AccountID: a.AccountID,
	}
	res, err := a.accounts.utxoDB.Reserve(ctx, src, a.Amount, a.ClientToken, b.MaxTime())
	if err != nil {
		return errors.Wrap(err, "reserving utxos")
	}

	// Cancel the reservation if the build gets rolled back.
	b.OnRollback(canceler(ctx, a.accounts, res.ID))

	for _, r := range res.UTXOs {
		txInput, sigInst, err := utxoToInputs(ctx, acct, r, a.ReferenceData)
		if err != nil {
			return errors.Wrap(err, "creating inputs")
		}
		err = b.AddInput(txInput, sigInst)
		if err != nil {
			return errors.Wrap(err, "adding inputs")
		}
	}

	if res.Change > 0 {
		acp, err := a.accounts.createControlProgram(ctx, a.AccountID, true, b.MaxTime())
		if err != nil {
			return errors.Wrap(err, "creating control program")
		}

		// Don't insert the control program until callbacks are executed.
		a.accounts.insertControlProgramDelayed(ctx, b, acp)

		err = b.AddOutput(bc.NewTxOutput(a.AssetID, res.Change, acp.controlProgram, nil))
		if err != nil {
			return errors.Wrap(err, "adding change output")
		}
	}
	return nil
}

func (m *Manager) NewSpendUTXOAction(outputID bc.OutputID) txbuilder.Action {
	return &spendUTXOAction{
		accounts: m,
		OutputID: &outputID,
	}
}

func (m *Manager) DecodeSpendUTXOAction(data []byte) (txbuilder.Action, error) {
	a := &spendUTXOAction{accounts: m}
	err := json.Unmarshal(data, a)
	return a, err
}

type spendUTXOAction struct {
	accounts *Manager
	OutputID *bc.OutputID `json:"output_id"`
	TxHash   *bc.Hash     `json:"transaction_id"`
	TxOut    *uint32      `json:"position"`

	ReferenceData chainjson.Map `json:"reference_data"`
	ClientToken   *string       `json:"client_token"`
}

func (a *spendUTXOAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var outid bc.OutputID

	if a.OutputID != nil {
		outid = *a.OutputID
	} else if a.TxHash != nil && a.TxOut != nil {
		// This is compatibility layer - legacy apps can spend outputs via the raw <txid:index> pair.
		q := `SELECT output_id FROM account_utxos WHERE tx_hash=$1 AND index=$2`
		err := a.accounts.utxoDB.db.QueryRow(ctx, q, *a.TxHash, *a.TxOut).Scan(&outid)
		if err == sql.ErrNoRows {
			return pg.ErrUserInputNotFound
		} else if err != nil {
			return err
		}
	} else {
		// Note: here we do not attempt to check if txid is present, but position is missing, or vice versa.
		// Instead, the user has to update their code to use the new API anyway.
		return txbuilder.MissingFieldsError("output_id")
	}

	res, err := a.accounts.utxoDB.ReserveUTXO(ctx, outid, a.ClientToken, b.MaxTime())
	if err != nil {
		return err
	}
	b.OnRollback(canceler(ctx, a.accounts, res.ID))

	acct, err := a.accounts.findByID(ctx, res.Source.AccountID)
	if err != nil {
		return err
	}
	txInput, sigInst, err := utxoToInputs(ctx, acct, res.UTXOs[0], a.ReferenceData)
	if err != nil {
		return err
	}
	return b.AddInput(txInput, sigInst)
}

// Best-effort cancellation attempt to put in txbuilder.BuildResult.Rollback.
func canceler(ctx context.Context, m *Manager, rid uint64) func() {
	return func() {
		err := m.utxoDB.Cancel(ctx, rid)
		if err != nil {
			log.Error(ctx, err)
		}
	}
}

func utxoToInputs(ctx context.Context, account *signers.Signer, u *utxo, refData []byte) (
	*bc.TxInput,
	*txbuilder.SigningInstruction,
	error,
) {
	txInput := bc.NewSpendInput(u.OutputID, nil, u.AssetID, u.Amount, u.ControlProgram, refData)

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

func (a *controlAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string
	if a.AccountID == "" {
		missing = append(missing, "account_id")
	}
	if a.AssetID == (bc.AssetID{}) {
		missing = append(missing, "asset_id")
	}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	// Produce a control program, but don't insert it into the database yet.
	acp, err := a.accounts.createControlProgram(ctx, a.AccountID, false, b.MaxTime())
	if err != nil {
		return err
	}
	a.accounts.insertControlProgramDelayed(ctx, b, acp)

	return b.AddOutput(bc.NewTxOutput(a.AssetID, a.Amount, acp.controlProgram, a.ReferenceData))
}

// insertControlProgramDelayed takes a template builder and an account
// control program that hasn't been inserted to the database yet. It
// registers callbacks on the TemplateBuilder so that all of the template's
// account control programs are batch inserted if building the rest of
// the template is successful.
func (m *Manager) insertControlProgramDelayed(ctx context.Context, b *txbuilder.TemplateBuilder, acp *controlProgram) {
	m.delayedACPsMu.Lock()
	m.delayedACPs[b] = append(m.delayedACPs[b], acp)
	m.delayedACPsMu.Unlock()

	b.OnRollback(func() {
		m.delayedACPsMu.Lock()
		delete(m.delayedACPs, b)
		m.delayedACPsMu.Unlock()
	})
	b.OnBuild(func() error {
		m.delayedACPsMu.Lock()
		acps := m.delayedACPs[b]
		delete(m.delayedACPs, b)
		m.delayedACPsMu.Unlock()

		// Insert all of the account control programs at once.
		if len(acps) == 0 {
			return nil
		}
		return m.insertAccountControlProgram(ctx, acps...)
	})
}
