package account

import (
	"context"
	"math/rand"
	"time"

	"chain/core/account/utxodb"
	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
)

type SpendAction struct {
	Params struct {
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
	}
	Constraints   txbuilder.ConstraintList
	ReferenceData json.Map `json:"reference_data"`
	ClientToken   *string  `json:"client_token"`
}

// TTL returns the time-to-live of the reservation created by this action.
func (a *SpendAction) TTL() time.Duration {
	ttl := a.Params.TTL
	if ttl == 0 {
		ttl = time.Minute
	}
	return ttl
}

func (a *SpendAction) Build(ctx context.Context, maxTime time.Time) ([]*bc.TxInput, []*bc.TxOutput, []*txbuilder.Input, error) {
	acct, err := FindByID(ctx, a.Params.AccountID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "get account info")
	}

	utxodbSource := utxodb.Source{
		AssetID:     a.Params.AssetID,
		Amount:      a.Params.Amount,
		AccountID:   a.Params.AccountID,
		TxHash:      a.Params.TxHash,
		OutputIndex: a.Params.TxOut,
		ClientToken: a.ClientToken,
	}
	utxodbSources := []utxodb.Source{utxodbSource}
	reserved, change, err := utxodb.Reserve(ctx, utxodbSources, maxTime)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "reserving utxos")
	}

	var (
		txins      []*bc.TxInput
		tplIns     []*txbuilder.Input
		changeOuts []*bc.TxOutput
	)

	constraints := a.Constraints
	if len(constraints) > 0 {
		// Add constraints only if some are already specified. If none
		// are, leave the constraint list empty to get the default
		// commit-to-txsighash behavior.
		constraints = append(constraints, txbuilder.TTLConstraint(maxTime))
	}

	for _, r := range reserved {
		txInput, templateInput, err := utxoToInputs(ctx, acct, r, a.ReferenceData, constraints)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "creating inputs")
		}

		txins = append(txins, txInput)
		tplIns = append(tplIns, templateInput)
	}
	if len(change) > 0 {
		changeAmounts := breakupChange(change[0].Amount)

		// TODO(bobg): As pointed out by @kr, each time through this loop
		// involves a db write.
		// May be preferable performancewise to allocate all the
		// destinations in one call.
		for _, changeAmount := range changeAmounts {
			acp, err := CreateControlProgram(ctx, a.Params.AccountID)
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "creating control program")
			}
			changeOuts = append(changeOuts, bc.NewTxOutput(a.Params.AssetID, changeAmount, acp, nil))

			if len(constraints) > 0 {
				// Constrain every input to require this change output.

				for _, tplIn := range tplIns {
					if len(tplIn.WitnessComponents) != 1 {
						// shouldn't happen
						continue
					}
					if sw, ok := tplIn.WitnessComponents[0].(*txbuilder.SignatureWitness); ok {
						pc := &txbuilder.PayConstraint{
							AssetAmount: bc.AssetAmount{
								AssetID: a.Params.AssetID,
								Amount:  changeAmount,
							},
							Program: acp,
						}
						sw.Constraints = append(sw.Constraints, pc)
					}
				}
			}
		}
	}

	return txins, changeOuts, tplIns, nil
}

func breakupChange(total uint64) (amounts []uint64) {
	for total > 1 && rand.Intn(2) == 0 {
		thisChange := 1 + uint64(rand.Int63n(int64(total)))
		amounts = append(amounts, thisChange)
		total -= thisChange
	}
	if total > 0 {
		amounts = append(amounts, total)
	}
	return amounts
}

type SpendUTXOAction struct {
	Params struct {
		TxHash bc.Hash       `json:"transaction_id"`
		TxOut  uint32        `json:"position"`
		TTL    time.Duration `json:"reservation_ttl"`
	}
	Constraints   txbuilder.ConstraintList
	ReferenceData json.Map `json:"reference_data"`
	ClientToken   *string  `json:"client_token"`
}

// TTL returns the time-to-live of the reservation created by this action.
func (a *SpendUTXOAction) TTL() time.Duration {
	ttl := a.Params.TTL
	if ttl == 0 {
		ttl = time.Minute
	}
	return ttl
}

func (a *SpendUTXOAction) Build(ctx context.Context, maxTime time.Time) ([]*bc.TxInput, []*bc.TxOutput, []*txbuilder.Input, error) {
	r, err := utxodb.ReserveUTXO(ctx, a.Params.TxHash, a.Params.TxOut, a.ClientToken, maxTime)
	if err != nil {
		return nil, nil, nil, err
	}

	acct, err := FindByID(ctx, r.AccountID)
	if err != nil {
		return nil, nil, nil, err
	}

	constraints := a.Constraints
	if len(constraints) > 0 {
		// Add constraints only if some are already specified. If none
		// are, leave the constraint list empty to get the default
		// commit-to-txsighash behavior.
		constraints = append(constraints, txbuilder.TTLConstraint(maxTime))
	}

	txInput, tplInput, err := utxoToInputs(ctx, acct, r, a.ReferenceData, constraints)
	if err != nil {
		return nil, nil, nil, err
	}

	return []*bc.TxInput{txInput}, nil, []*txbuilder.Input{tplInput}, nil
}

func utxoToInputs(ctx context.Context, account *Account, u *utxodb.UTXO, refData []byte, constraints []txbuilder.Constraint) (*bc.TxInput, *txbuilder.Input, error) {
	txInput := bc.NewSpendInput(u.Hash, u.Index, nil, u.AssetID, u.Amount, u.Script, refData)

	templateInput := &txbuilder.Input{
		AssetAmount: u.AssetAmount,
	}

	path := signers.Path(account.Signer, signers.AccountKeySpace, u.ControlProgramIndex[:])
	keyIDs := txbuilder.KeyIDs(account.XPubs, path)

	if len(constraints) > 0 {
		// Add constraints only if some are already specified. If none
		// are, leave the constraint list empty to get the default
		// commit-to-txsighash behavior.
		constraints = append(constraints, txbuilder.OutpointConstraint(u.Outpoint))
	}

	templateInput.AddWitnessKeys(keyIDs, account.Quorum, constraints)

	return txInput, templateInput, nil
}

type ControlAction struct {
	Params struct {
		bc.AssetAmount
		AccountID string `json:"account_id"`

		// These fields are only necessary for filtering
		// aliases on transaction build requests. A wrapper
		// function reads them to set the ID fields. They are
		// not used anywhere else in the code base.
		AccountAlias string `json:"account_alias"`
		AssetAlias   string `json:"asset_alias"`
	}
	ReferenceData json.Map `json:"reference_data"`
}

func (a *ControlAction) Build(ctx context.Context, _ time.Time) ([]*bc.TxInput, []*bc.TxOutput, []*txbuilder.Input, error) {
	acp, err := CreateControlProgram(ctx, a.Params.AccountID)
	if err != nil {
		return nil, nil, nil, err
	}
	out := bc.NewTxOutput(a.Params.AssetID, a.Params.Amount, acp, a.ReferenceData)
	return nil, []*bc.TxOutput{out}, nil, nil
}

// CancelReservations cancels any existing reservations
// for the given outpoints.
func CancelReservations(ctx context.Context, outpoints []bc.Outpoint) error {
	return utxodb.Cancel(ctx, outpoints)
}
