package account

import (
	"math/rand"
	"time"

	"golang.org/x/net/context"

	"chain/core/account/utxodb"
	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/crypto/ed25519/hd25519"
	"chain/encoding/json"
	"chain/errors"
)

type SpendAction struct {
	Params struct {
		bc.AssetAmount
		AccountID string        `json:"account_id"`
		TxHash    *bc.Hash      `json:"transaction_hash"`
		TxOut     *uint32       `json:"transaction_output"`
		TTL       time.Duration `json:"reservation_ttl"`
	}
	ReferenceData json.HexBytes `json:"reference_data"`
	ClientToken   *string       `json:"client_token"`
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
		txInput := bc.NewSpendInput(r.Hash, r.Index, nil, r.AssetID, r.Amount, r.Script, a.ReferenceData)

		templateInput := &txbuilder.Input{}
		inputAccount, err := Find(ctx, r.AccountID)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "get account info")
		}

		path := signers.Path(inputAccount, signers.AccountKeySpace, r.ControlProgramIndex[:])
		derivedXPubs := hd25519.DeriveXPubs(inputAccount.XPubs, path)
		derivedPKs := hd25519.XPubKeys(derivedXPubs)

		redeemScript, err := txscript.TxMultiSigScript(derivedPKs, inputAccount.Quorum)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "compute redeem script")
		}
		templateInput.AssetID = r.AssetID
		templateInput.Amount = r.Amount
		templateInput.AddWitnessSigs(txbuilder.InputSigs(inputAccount.XPubs, path), inputAccount.Quorum, nil)
		templateInput.AddWitnessData(redeemScript)

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

type ControlAction struct {
	Params struct {
		bc.AssetAmount
		AccountID string `json:"account_id"`
	}
	ReferenceData json.HexBytes `json:"reference_data"`
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
