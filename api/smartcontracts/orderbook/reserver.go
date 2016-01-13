package orderbook

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
)

type redeemReserver struct {
	openOrder     *OpenOrder
	offerAmount   uint64
	paymentAmount *bc.AssetAmount
}

// Reserve satisfies asset.Reserver.
func (reserver *redeemReserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*asset.ReserveResult, error) {
	openOrder := reserver.openOrder
	changeAmount, err := reserveOrder(ctx, openOrder, assetAmount.Amount, ttl)
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
	redeemScript, err := sb.Script()
	if err != nil {
		return nil, err
	}
	result := &asset.ReserveResult{
		Items: []*asset.ReserveResultItem{
			{
				TxInput:       &bc.TxInput{Previous: openOrder.Outpoint},
				TemplateInput: &asset.Input{RedeemScript: redeemScript},
			},
		},
	}
	if changeAmount > 0 {
		changeAssetAmount := &bc.AssetAmount{
			AssetID: assetAmount.AssetID,
			Amount:  changeAmount,
		}
		result.Change, err = NewDestinationWithScript(ctx, changeAssetAmount, &openOrder.OrderInfo, true, nil, openOrder.Script)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// NewRedeemSource creates an asset.Source that redeems a specific
// Orderbook order by paying one of its requested prices.
func NewRedeemSource(openOrder *OpenOrder, offerAmount uint64, paymentAmount *bc.AssetAmount) *asset.Source {
	return &asset.Source{
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

func (reserver *cancelReserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*asset.ReserveResult, error) {
	openOrder := reserver.openOrder
	changeAmount, err := reserveOrder(ctx, openOrder, assetAmount.Amount, ttl)
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
	redeemScript, err := sb.Script()
	if err != nil {
		return nil, err
	}
	result := &asset.ReserveResult{
		Items: []*asset.ReserveResultItem{
			{
				TxInput: &bc.TxInput{Previous: openOrder.Outpoint},
				TemplateInput: &asset.Input{
					RedeemScript: redeemScript,
					SignScript:   contractScript,
					Sigs:         inputSigs(hdkey.Derive(sellerAddr.Keys, appdb.ReceiverPath(sellerAddr, sellerAddr.Index))),
				},
			},
		},
	}
	return result, nil
}

func reserveOrder(ctx context.Context, openOrder *OpenOrder, amount uint64, ttl time.Duration) (changeAmount uint64, err error) {
	// TODO(bobg): Consider verifying that a matching record exists in the orderbook_utxos table.
	const q = `
		UPDATE utxos SET reserved_until = $1
		    WHERE tx_hash = $2 AND index = $3 AND reserved_until < NOW()
		        AND NOT EXISTS (SELECT 1 FROM pool_inputs WHERE tx_hash = $2 AND index = $3)
	`

	now := time.Now().UTC()
	expiry := now.Add(ttl)
	result, err := pg.FromContext(ctx).Exec(ctx, q, expiry, openOrder.Outpoint.Hash, openOrder.Outpoint.Index)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected == 0 {
		return 0, fmt.Errorf("utxo not found: %s", openOrder.Outpoint.Hash)
	}

	if amount < openOrder.Amount {
		changeAmount = openOrder.Amount - amount
	}

	return changeAmount, nil
}

// NewCancelSource creates an asset.Source that cancels a specific
// Orderbook order, sending its balance back to the seller.
func NewCancelSource(openOrder *OpenOrder) *asset.Source {
	return &asset.Source{
		AssetAmount: openOrder.AssetAmount,
		Reserver: &cancelReserver{
			openOrder: openOrder,
		},
	}
}

// TODO(bobg): refactor to not duplicate this function from
// api/asset/asset.go
func inputSigs(keys []*hdkey.Key) (sigs []*asset.Signature) {
	for _, k := range keys {
		sigs = append(sigs, &asset.Signature{
			XPub:           k.Root.String(),
			DerivationPath: k.Path,
		})
	}
	return sigs
}
