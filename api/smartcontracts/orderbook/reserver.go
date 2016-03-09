package orderbook

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txbuilder"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/hdkey"
	"chain/fedchain/txscript"
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
		return nil, err
	}
	contractScript, err := buildContract(len(openOrder.Prices))
	if err != nil {
		return nil, err
	}
	sb := txscript.NewScriptBuilder()
	sb = sb.AddInt64(int64(reserver.paymentAmount.Amount))
	sb = sb.AddInt64(int64(changeAmount))
	sb = sb.AddInt64(1)
	sb = sb.AddData(contractScript)
	sigScriptSuffix, err := sb.Script()
	if err != nil {
		return nil, err
	}
	result := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: &bc.TxInput{Previous: openOrder.Outpoint},
				TemplateInput: &txbuilder.Input{
					AssetAmount:     openOrder.AssetAmount,
					SigScriptSuffix: sigScriptSuffix,
				},
			},
		},
	}
	if changeAmount > 0 {
		changeAssetAmount := &bc.AssetAmount{
			AssetID: assetAmount.AssetID,
			Amount:  changeAmount,
		}
		result.Change, err = NewDestinationWithScript(ctx, changeAssetAmount, &openOrder.OrderInfo, nil, openOrder.Script)
		if err != nil {
			return nil, err
		}
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
	sb := txscript.NewScriptBuilder()
	sb = sb.AddData(sellerAddr.RedeemScript).AddInt64(0).AddData(contractScript)
	sigScriptSuffix, err := sb.Script()
	if err != nil {
		return nil, err
	}
	result := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: &bc.TxInput{Previous: openOrder.Outpoint},
				TemplateInput: &txbuilder.Input{
					AssetAmount:     openOrder.AssetAmount,
					SigScriptSuffix: sigScriptSuffix,
					Sigs:            inputSigs(hdkey.Derive(sellerAddr.Keys, appdb.ReceiverPath(sellerAddr, sellerAddr.Index))),
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

// TODO(bobg): refactor to not duplicate this function from
// api/asset/asset.go
func inputSigs(keys []*hdkey.Key) (sigs []*txbuilder.Signature) {
	for _, k := range keys {
		sigs = append(sigs, &txbuilder.Signature{
			XPub:           k.Root.String(),
			DerivationPath: k.Path,
		})
	}
	return sigs
}
