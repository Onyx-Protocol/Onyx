package account

import (
	"context"
	"math/rand"
	"time"

	"chain/core/account/utxodb"
	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/hd25519"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vmutil"
)

type SpendAction struct {
	Params struct {
		bc.AssetAmount
		AccountID string        `json:"account_id"`
		TxHash    *bc.Hash      `json:"transaction_id"`
		TxOut     *uint32       `json:"position"`
		TTL       time.Duration `json:"reservation_ttl"`
	}
	ReferenceData json.Map `json:"reference_data"`
	ClientToken   *string  `json:"client_token"`
}

func (a *SpendAction) Build(ctx context.Context) ([]*bc.TxInput, []*bc.TxOutput, []*txbuilder.Input, error) {
	ttl := a.Params.TTL
	if ttl == 0 {
		ttl = time.Minute
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
	reserved, change, err := utxodb.Reserve(ctx, utxodbSources, ttl)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		txins      []*bc.TxInput
		tplIns     []*txbuilder.Input
		changeOuts []*bc.TxOutput
	)
	for _, r := range reserved {
		txInput, templateInput, err := utxoToInputs(ctx, r, a.ReferenceData)
		if err != nil {
			return nil, nil, nil, err
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
				return nil, nil, nil, err
			}
			changeOuts = append(changeOuts, bc.NewTxOutput(a.Params.AssetID, changeAmount, acp, nil))
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
	ReferenceData json.Map `json:"reference_data"`
	ClientToken   *string  `json:"client_token"`
}

func (a *SpendUTXOAction) Build(ctx context.Context) ([]*bc.TxInput, []*bc.TxOutput, []*txbuilder.Input, error) {
	ttl := a.Params.TTL
	if ttl == 0 {
		ttl = time.Minute
	}

	r, err := utxodb.ReserveUTXO(ctx, a.Params.TxHash, a.Params.TxOut, a.ClientToken, a.Params.TTL)
	if err != nil {
		return nil, nil, nil, err
	}

	txInput, tplInput, err := utxoToInputs(ctx, r, a.ReferenceData)
	if err != nil {
		return nil, nil, nil, err
	}

	return []*bc.TxInput{txInput}, nil, []*txbuilder.Input{tplInput}, nil
}

func utxoToInputs(ctx context.Context, u *utxodb.UTXO, refData []byte) (*bc.TxInput, *txbuilder.Input, error) {
	txInput := bc.NewSpendInput(u.Hash, u.Index, nil, u.AssetID, u.Amount, u.Script, refData)

	templateInput := &txbuilder.Input{}
	inputAccount, err := FindByID(ctx, u.AccountID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "get account info")
	}

	path := signers.Path(inputAccount.Signer, signers.AccountKeySpace, u.ControlProgramIndex[:])
	derivedXPubs := hd25519.DeriveXPubs(inputAccount.XPubs, path)
	derivedPKs := hd25519.XPubKeys(derivedXPubs)

	redeemScript, err := vmutil.TxMultiSigScript(derivedPKs, inputAccount.Quorum)
	if err != nil {
		return nil, nil, errors.Wrap(err, "compute redeem script")
	}
	templateInput.AssetID = u.AssetID
	templateInput.Amount = u.Amount
	templateInput.AddWitnessSigs(txbuilder.InputSigs(inputAccount.XPubs, path), inputAccount.Quorum, nil)
	templateInput.AddWitnessData(redeemScript)

	return txInput, templateInput, nil
}

type ControlAction struct {
	Params struct {
		bc.AssetAmount
		AccountID string `json:"account_id"`
	}
	ReferenceData json.Map `json:"reference_data"`
}

func (a *ControlAction) Build(ctx context.Context) ([]*bc.TxInput, []*bc.TxOutput, []*txbuilder.Input, error) {
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
