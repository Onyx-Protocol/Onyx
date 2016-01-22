package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txbuilder"
	"chain/api/utxodb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
)

type AccountReserver struct {
	AccountID string
}

func (reserver *AccountReserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	utxodbSource := utxodb.Source{
		AssetID:   assetAmount.AssetID,
		Amount:    assetAmount.Amount,
		AccountID: reserver.AccountID,
	}
	utxodbSources := []utxodb.Source{utxodbSource}
	reserved, change, err := utxodb.Reserve(ctx, utxodbSources, ttl)
	if err != nil {
		return nil, err
	}

	result := &txbuilder.ReserveResult{}
	for _, r := range reserved {
		txInput := &bc.TxInput{
			Previous: r.Outpoint,
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
		changeAssetAmount := &bc.AssetAmount{
			AssetID: assetAmount.AssetID,
			Amount:  change[0].Amount,
		}
		result.Change, err = NewAccountDestination(ctx, changeAssetAmount, reserver.AccountID, nil)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func NewAccountSource(ctx context.Context, assetAmount *bc.AssetAmount, accountID string) *txbuilder.Source {
	return &txbuilder.Source{
		AssetAmount: *assetAmount,
		Reserver: &AccountReserver{
			AccountID: accountID,
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
