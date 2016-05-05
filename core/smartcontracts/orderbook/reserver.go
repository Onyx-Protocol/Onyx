package orderbook

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/cos/txscript"
	"chain/database/pg"
	"chain/errors"
)

type redeemReserver struct {
	openOrder     *OpenOrder
	offerAmount   uint64
	paymentAmount *bc.AssetAmount
}

// Reserve satisfies txbuilder.Reserver.
func (reserver *redeemReserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	openOrder := reserver.openOrder
	changeAmount, err := reserveOrder(ctx, openOrder, assetAmount.Amount)
	if err != nil {
		return nil, errors.Wrap(err, "reserving order")
	}
	contractScript, err := buildContract(len(openOrder.Prices))
	if err != nil {
		return nil, errors.Wrap(err, "building contract")
	}

	inputs := []txscript.Item{
		txscript.NumItem(reserver.paymentAmount.Amount),
		txscript.NumItem(changeAmount),
		txscript.NumItem(1),
	}
	sigscript, err := txscript.RedeemP2C(openOrder.Script, contractScript, inputs)
	if err != nil {
		return nil, errors.Wrap(err, "building sigscript")
	}

	if err != nil {
		return nil, err
	}
	result := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: &bc.TxInput{
					Previous:    openOrder.Outpoint,
					AssetAmount: openOrder.AssetAmount,
					PrevScript:  openOrder.Script,
				},
				TemplateInput: &txbuilder.Input{
					AssetAmount:     openOrder.AssetAmount,
					SigScriptSuffix: sigscript,
				},
			},
		},
	}
	if changeAmount > 0 {
		changeAssetAmount := &bc.AssetAmount{
			AssetID: assetAmount.AssetID,
			Amount:  changeAmount,
		}
		changeDest, err := NewDestinationWithScript(ctx, changeAssetAmount, &openOrder.OrderInfo, nil, openOrder.Script)
		if err != nil {
			return nil, err
		}
		result.Change = append(result.Change, changeDest)
	}
	return result, nil
}

// NewRedeemSource creates an txbuilder.Source that redeems a specific
// Orderbook order by paying one of its requested prices.
func NewRedeemSource(openOrder *OpenOrder, offerAmount uint64, paymentAmount *bc.AssetAmount) *txbuilder.Source {
	return &txbuilder.Source{
		AssetAmount: bc.AssetAmount{
			AssetID: openOrder.AssetID,
			Amount:  offerAmount,
		},
		Reserver: &redeemReserver{
			openOrder:     openOrder,
			offerAmount:   offerAmount,
			paymentAmount: paymentAmount,
		},
	}
}

type cancelReserver struct {
	openOrder *OpenOrder
}

// cancelReserver error.
var ErrUnexpectedChange = errors.New("unexpected change")

func (reserver *cancelReserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	openOrder := reserver.openOrder
	changeAmount, err := reserveOrder(ctx, openOrder, assetAmount.Amount)
	if err != nil {
		return nil, err
	}
	if changeAmount > 0 {
		return nil, ErrUnexpectedChange
	}
	contractScript, err := buildContract(len(openOrder.Prices))
	if err != nil {
		return nil, err
	}
	sellerScript, err := openOrder.SellerScript()
	if err != nil {
		return nil, err
	}
	sellerAddr, err := appdb.GetAddress(ctx, sellerScript)
	if err != nil {
		sellerScriptStr, _ := txscript.DisasmString(sellerScript)
		return nil, errors.Wrapf(err, "could not get address for seller script [%s]", sellerScriptStr)
	}
	inputs := []txscript.Item{
		txscript.DataItem(sellerAddr.RedeemScript),
		txscript.NumItem(0),
	}
	sigscript, err := txscript.RedeemP2C(openOrder.Script, contractScript, inputs)
	if err != nil {
		return nil, err
	}
	result := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: &bc.TxInput{
					Previous:    openOrder.Outpoint,
					AssetAmount: openOrder.AssetAmount,
					PrevScript:  openOrder.Script,
				},
				TemplateInput: &txbuilder.Input{
					AssetAmount:     openOrder.AssetAmount,
					SigScriptSuffix: sigscript,
					Sigs:            txbuilder.InputSigs(hdkey.Derive(sellerAddr.Keys, appdb.ReceiverPath(sellerAddr, sellerAddr.Index))),
				},
			},
		},
	}
	return result, nil
}

func reserveOrder(ctx context.Context, openOrder *OpenOrder, amount uint64) (changeAmount uint64, err error) {
	const q = `
		SELECT COUNT(*) FROM utxos
		  WHERE (tx_hash, index) = ($1, $2)
		  AND (tx_hash, index) NOT IN (TABLE pool_inputs)
	`

	var cnt int
	row := pg.QueryRow(ctx, q, openOrder.Outpoint.Hash, openOrder.Outpoint.Index)
	err = row.Scan(&cnt)
	if err != nil {
		return 0, err
	}

	if cnt == 0 {
		return 0, fmt.Errorf("utxo not found: %s", openOrder.Outpoint.Hash)
	}

	if amount < openOrder.Amount {
		changeAmount = openOrder.Amount - amount
	}

	return changeAmount, nil
}

// NewCancelSource creates an txbuilder.Source that cancels a specific
// Orderbook order, sending its balance back to the seller.
func NewCancelSource(openOrder *OpenOrder) *txbuilder.Source {
	return &txbuilder.Source{
		AssetAmount: openOrder.AssetAmount,
		Reserver: &cancelReserver{
			openOrder: openOrder,
		},
	}
}
