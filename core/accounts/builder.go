package accounts

import (
	"math/rand"
	"time"

	"golang.org/x/net/context"

	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/core/utxodb"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/crypto/ed25519/hd25519"
	"chain/errors"
)

// Reserver selects utxos that are sent to control programs contained
// in the account. It satisfies the txbuilder.Reserver interface.
type Reserver struct {
	accountID   string
	txHash      *bc.Hash // optional filter
	outputIndex *uint32  // optional filter
	clientToken *string
}

// Reserve selects utxos held in the account.
func (reserver *Reserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	utxodbSource := utxodb.Source{
		AssetID:     assetAmount.AssetID,
		Amount:      assetAmount.Amount,
		AccountID:   reserver.accountID,
		TxHash:      reserver.txHash,
		OutputIndex: reserver.outputIndex,
		ClientToken: reserver.clientToken,
	}
	utxodbSources := []utxodb.Source{utxodbSource}
	reserved, change, err := utxodb.Reserve(ctx, utxodbSources, ttl)
	if err != nil {
		return nil, err
	}

	result := &txbuilder.ReserveResult{}
	for _, r := range reserved {
		txInput := bc.NewSpendInput(r.Hash, r.Index, nil, r.AssetID, r.Amount, r.Script, nil)

		templateInput := &txbuilder.Input{}
		inputAccount, err := Find(ctx, r.AccountID)
		if err != nil {
			return nil, errors.Wrap(err, "get account info")
		}

		path := signers.Path(inputAccount, signers.AccountKeySpace, r.ControlProgramIndex[:])
		derivedXPubs := hd25519.DeriveXPubs(inputAccount.XPubs, path)
		derivedPKs := hd25519.XPubKeys(derivedXPubs)

		redeemScript, err := txscript.MultiSigScript(derivedPKs, inputAccount.Quorum)
		if err != nil {
			return nil, errors.Wrap(err, "compute redeem script")
		}
		templateInput.AssetID = r.AssetID
		templateInput.Amount = r.Amount
		templateInput.AddWitnessSigs(txbuilder.InputSigs(inputAccount.XPubs, path), inputAccount.Quorum, nil)
		templateInput.AddWitnessData(redeemScript)

		item := &txbuilder.ReserveResultItem{
			TxInput:       txInput,
			TemplateInput: templateInput,
		}

		result.Items = append(result.Items, item)
	}
	if len(change) > 0 {
		changeAmounts := breakupChange(change[0].Amount)

		// TODO(bobg): As pointed out by @kr, each time through this loop
		// involves a db write (in the call to NewDestination).
		// May be preferable performancewise to allocate all the
		// destinations in one call.
		for _, changeAmount := range changeAmounts {
			dest, err := NewDestination(ctx, &bc.AssetAmount{AssetID: assetAmount.AssetID, Amount: changeAmount}, reserver.accountID, nil)
			if err != nil {
				return nil, errors.Wrap(err, "creating change destination")
			}
			result.Change = append(result.Change, dest)
		}
	}

	return result, nil
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

// NewSource returns a new txbuilder.Source with an account reserver.
func NewSource(ctx context.Context, assetAmount *bc.AssetAmount, accountID string, txHash *bc.Hash, outputIndex *uint32, clientToken *string) *txbuilder.Source {
	return &txbuilder.Source{
		AssetAmount: *assetAmount,
		Reserver: &Reserver{
			accountID:   accountID,
			txHash:      txHash,
			outputIndex: outputIndex,
			clientToken: clientToken,
		},
	}
}

// Receiver satisfies the txbuilder.Receiver interface.
type Receiver struct {
	controlProgram []byte
	accountID      string
}

// PKScript returns a control program held by the account.
func (receiver *Receiver) PKScript() []byte { return receiver.controlProgram }

// AccountID returns the account id of the receiver.
func (receiver *Receiver) AccountID() string { return receiver.accountID }

// NewDestination returns a new txbuilder.Destination with an account receiver.
func NewDestination(ctx context.Context, assetAmount *bc.AssetAmount, accountID string, metadata []byte) (*txbuilder.Destination, error) {
	acp, err := CreateControlProgram(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return &txbuilder.Destination{
		AssetAmount: *assetAmount,
		Metadata:    metadata,
		Receiver:    &Receiver{controlProgram: acp, accountID: accountID},
	}, nil
}

// CancelReservations cancels any existing reservations
// for the given outpoints.
func CancelReservations(ctx context.Context, outpoints []bc.Outpoint) error {
	return utxodb.Cancel(ctx, outpoints)
}
