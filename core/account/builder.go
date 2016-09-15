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

// TTL returns the time-to-live of the reservation created by this action.
func (a *SpendAction) GetTTL() time.Duration {
	ttl := a.TTL
	if ttl == 0 {
		ttl = time.Minute
	}
	return ttl
}

func (a *SpendAction) Build(ctx context.Context, maxTime time.Time) (
	[]*bc.TxInput,
	[]*bc.TxOutput,
	[]*txbuilder.SigningInstruction,
	error,
) {
	acct, err := FindByID(ctx, a.AccountID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "get account info")
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
		return nil, nil, nil, errors.Wrap(err, "reserving utxos")
	}

	var (
		txins      []*bc.TxInput
		tplInsts   []*txbuilder.SigningInstruction
		changeOuts []*bc.TxOutput
	)

	for _, r := range reserved {
		txInput, sigInst, err := utxoToInputs(ctx, acct, r, a.ReferenceData)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "creating inputs")
		}

		txins = append(txins, txInput)
		tplInsts = append(tplInsts, sigInst)
	}
	if len(change) > 0 {
		changeAmounts := breakupChange(change[0].Amount)

		// TODO(bobg): As pointed out by @kr, each time through this loop
		// involves a db write.
		// May be preferable performancewise to allocate all the
		// destinations in one call.
		for _, changeAmount := range changeAmounts {
			acp, err := CreateControlProgram(ctx, a.AccountID)
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "creating control program")
			}
			changeOuts = append(changeOuts, bc.NewTxOutput(a.AssetID, changeAmount, acp, nil))
		}
	}

	return txins, changeOuts, tplInsts, nil
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
	TxHash bc.Hash       `json:"transaction_id"`
	TxOut  uint32        `json:"position"`
	TTL    time.Duration `json:"reservation_ttl"`

	ReferenceData json.Map `json:"reference_data"`
	ClientToken   *string  `json:"client_token"`
}

// TTL returns the time-to-live of the reservation created by this action.
func (a *SpendUTXOAction) GetTTL() time.Duration {
	ttl := a.TTL
	if ttl == 0 {
		ttl = time.Minute
	}
	return ttl
}

func (a *SpendUTXOAction) Build(ctx context.Context, maxTime time.Time) (
	[]*bc.TxInput,
	[]*bc.TxOutput,
	[]*txbuilder.SigningInstruction,
	error,
) {
	r, err := utxodb.ReserveUTXO(ctx, a.TxHash, a.TxOut, a.ClientToken, maxTime)
	if err != nil {
		return nil, nil, nil, err
	}

	acct, err := FindByID(ctx, r.AccountID)
	if err != nil {
		return nil, nil, nil, err
	}

	txInput, sigInst, err := utxoToInputs(ctx, acct, r, a.ReferenceData)
	if err != nil {
		return nil, nil, nil, err
	}

	return []*bc.TxInput{txInput}, nil, []*txbuilder.SigningInstruction{sigInst}, nil
}

func utxoToInputs(ctx context.Context, account *Account, u *utxodb.UTXO, refData []byte) (
	*bc.TxInput,
	*txbuilder.SigningInstruction,
	error,
) {
	txInput := bc.NewSpendInput(u.Hash, u.Index, nil, u.AssetID, u.Amount, u.Script, refData)

	sigInst := &txbuilder.SigningInstruction{
		AssetAmount: u.AssetAmount,
	}

	path := signers.Path(account.Signer, signers.AccountKeySpace, u.ControlProgramIndex[:])
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

func (a *ControlAction) Build(ctx context.Context, _ time.Time) (
	[]*bc.TxInput,
	[]*bc.TxOutput,
	[]*txbuilder.SigningInstruction,
	error,
) {
	acp, err := CreateControlProgram(ctx, a.AccountID)
	if err != nil {
		return nil, nil, nil, err
	}
	out := bc.NewTxOutput(a.AssetID, a.Amount, acp, a.ReferenceData)
	return nil, []*bc.TxOutput{out}, nil, nil
}

// CancelReservations cancels any existing reservations
// for the given outpoints.
func CancelReservations(ctx context.Context, outpoints []bc.Outpoint) error {
	return utxodb.Cancel(ctx, outpoints)
}
