package account

import (
	"context"

	"chain/core/pb"
	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/errors"
	"chain/log"
	"chain/protocol/bc"
)

func (m *Manager) NewSpendAction(amt bc.AssetAmount, accountID string, refData []byte, clientToken *string) txbuilder.Action {
	return &spendAction{
		accounts:      m,
		AssetAmount:   amt,
		AccountID:     accountID,
		ReferenceData: refData,
		ClientToken:   clientToken,
	}
}

func (m *Manager) DecodeSpendAction(proto *pb.Action_SpendAccount) (txbuilder.Action, error) {
	assetID, err := bc.AssetIDFromBytes(proto.Asset.GetAssetId())
	if err != nil {
		return nil, errors.Wrap(err)
	}
	var ct *string
	if proto.ClientToken != "" {
		ct = &proto.ClientToken
	}
	return &spendAction{
		accounts:      m,
		AssetAmount:   bc.AssetAmount{AssetID: assetID, Amount: proto.Amount},
		AccountID:     proto.Account.GetAccountId(),
		ReferenceData: proto.ReferenceData,
		ClientToken:   ct,
	}, nil
}

type spendAction struct {
	accounts *Manager
	bc.AssetAmount
	AccountID     string
	ReferenceData []byte
	ClientToken   *string
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
		acp, err := a.accounts.createControlProgram(ctx, a.AccountID, true)
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

func (m *Manager) NewSpendUTXOAction(outpoint bc.Outpoint) txbuilder.Action {
	return &spendUTXOAction{
		accounts: m,
		TxHash:   &outpoint.Hash,
		TxOut:    &outpoint.Index,
	}
}

func (m *Manager) DecodeSpendUTXOAction(proto *pb.Action_SpendAccountUnspentOutput) (txbuilder.Action, error) {
	txhash, err := bc.HashFromBytes(proto.TxId)
	if err != nil {
		return nil, err
	}

	var ct *string
	if proto.ClientToken != "" {
		ct = &proto.ClientToken
	}

	return &spendUTXOAction{
		accounts:      m,
		TxHash:        &txhash,
		TxOut:         &proto.Position,
		ReferenceData: proto.ReferenceData,
		ClientToken:   ct,
	}, nil
}

type spendUTXOAction struct {
	accounts *Manager
	TxHash   *bc.Hash
	TxOut    *uint32

	ReferenceData []byte
	ClientToken   *string
}

func (a *spendUTXOAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string
	if a.TxHash == nil {
		missing = append(missing, "transaction_id")
	}
	if a.TxOut == nil {
		missing = append(missing, "position")
	}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	out := bc.Outpoint{Hash: *a.TxHash, Index: *a.TxOut}
	res, err := a.accounts.utxoDB.ReserveUTXO(ctx, out, a.ClientToken, b.MaxTime())
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
	*pb.TxTemplate_SigningInstruction,
	error,
) {
	txInput := bc.NewSpendInput(u.Hash, u.Index, nil, u.AssetID, u.Amount, u.ControlProgram, refData)

	sigInst := &pb.TxTemplate_SigningInstruction{
		AssetId: u.AssetID[:],
		Amount:  u.Amount,
	}

	path := signers.Path(account, signers.AccountKeySpace, u.ControlProgramIndex)

	sigInst.WitnessComponents = append(sigInst.WitnessComponents, pb.SignatureWitness(account.XPubs, path, account.Quorum))

	return txInput, sigInst, nil
}

func (m *Manager) NewControlAction(amt bc.AssetAmount, accountID string, refData []byte) txbuilder.Action {
	return &controlAction{
		accounts:      m,
		AssetAmount:   amt,
		AccountID:     accountID,
		ReferenceData: refData,
	}
}

func (m *Manager) DecodeControlAction(proto *pb.Action_ControlAccount) (txbuilder.Action, error) {
	assetID, err := bc.AssetIDFromBytes(proto.Asset.GetAssetId())
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return &controlAction{
		accounts:      m,
		AssetAmount:   bc.AssetAmount{AssetID: assetID, Amount: proto.Amount},
		AccountID:     proto.Account.GetAccountId(),
		ReferenceData: proto.ReferenceData,
	}, nil
}

type controlAction struct {
	accounts *Manager
	bc.AssetAmount
	AccountID     string
	ReferenceData []byte
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
	acp, err := a.accounts.createControlProgram(ctx, a.AccountID, false)
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
