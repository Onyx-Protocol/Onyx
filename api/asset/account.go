package asset

import (
	"math/rand"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txbuilder"
	"chain/api/utxodb"
	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/cos/txscript"
	"chain/errors"
)

type AccountReserver struct {
	AccountID   string
	TxHash      *bc.Hash // optional filter
	ClientToken *string
}

func (reserver *AccountReserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	utxodbSource := utxodb.Source{
		AssetID:     assetAmount.AssetID,
		Amount:      assetAmount.Amount,
		AccountID:   reserver.AccountID,
		TxHash:      reserver.TxHash,
		ClientToken: reserver.ClientToken,
	}
	utxodbSources := []utxodb.Source{utxodbSource}
	reserved, change, err := utxodb.Reserve(ctx, utxodbSources, ttl)
	if err != nil {
		return nil, err
	}

	result := &txbuilder.ReserveResult{}
	for _, r := range reserved {
		txInput := &bc.TxInput{
			Previous:    r.Outpoint,
			AssetAmount: r.AssetAmount,
			PrevScript:  r.Script,
		}

		templateInput := &txbuilder.Input{}
		addrInfo, err := appdb.AddrInfo(ctx, r.AccountID)
		if err != nil {
			return nil, errors.Wrap(err, "get addr info")
		}
		signers := hdkey.Derive(addrInfo.Keys, appdb.ReceiverPath(addrInfo, r.AddrIndex[:]))
		redeemScript, err := hdkey.RedeemScript(signers, addrInfo.SigsRequired)
		if err != nil {
			return nil, errors.Wrap(err, "compute redeem script")
		}
		templateInput.AssetID = r.AssetID
		templateInput.Amount = r.Amount
		templateInput.SigScriptSuffix = txscript.AddDataToScript(nil, redeemScript)
		templateInput.Sigs = txbuilder.InputSigs(signers)

		item := &txbuilder.ReserveResultItem{
			TxInput:       txInput,
			TemplateInput: templateInput,
		}

		result.Items = append(result.Items, item)
	}
	if len(change) > 0 {
		changeAmounts := breakupChange(change[0].Amount)

		// TODO(bobg): As pointed out by @kr, each time through this loop
		// involves a db write (in the call to NewAccountDestination).
		// May be preferable performancewise to allocate all the
		// destinations in one call.
		for _, changeAmount := range changeAmounts {
			dest, err := NewAccountDestination(ctx, &bc.AssetAmount{AssetID: assetAmount.AssetID, Amount: changeAmount}, reserver.AccountID, nil)
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

func NewAccountSource(ctx context.Context, assetAmount *bc.AssetAmount, accountID string, txHash *bc.Hash, clientToken *string) *txbuilder.Source {
	return &txbuilder.Source{
		AssetAmount: *assetAmount,
		Reserver: &AccountReserver{
			AccountID:   accountID,
			TxHash:      txHash,
			ClientToken: clientToken,
		},
	}
}

type AccountReceiver struct {
	addr *appdb.Address
}

func (receiver *AccountReceiver) PKScript() []byte { return receiver.addr.PKScript }

func NewAccountReceiver(addr *appdb.Address) *AccountReceiver {
	return &AccountReceiver{addr: addr}
}

func NewAccountDestination(ctx context.Context, assetAmount *bc.AssetAmount, accountID string, metadata []byte) (*txbuilder.Destination, error) {
	addr, err := appdb.NewAddress(ctx, accountID, true)
	if err != nil {
		return nil, err
	}
	receiver := NewAccountReceiver(addr)
	result := &txbuilder.Destination{
		AssetAmount: *assetAmount,
		Metadata:    metadata,
		Receiver:    receiver,
	}
	return result, nil
}

// CancelReservations cancels any existing reservations
// for the given outpoints.
func CancelReservations(ctx context.Context, outpoints []bc.Outpoint) error {
	return utxodb.Cancel(ctx, outpoints)
}
