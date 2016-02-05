package orderbook

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/crypto/hash160"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
)

type (
	// Price says pay PaymentAmount units of AssetID to get OfferAmount
	// units of the asset offered in an Orderbook contract.
	Price struct {
		// TODO(bobg): replace AssetID and PaymentAmount with an AssetAmount
		AssetID       bc.AssetID `json:"asset_id"`
		OfferAmount   uint64     `json:"offer_amount"`
		PaymentAmount uint64     `json:"payment_amount"`
	}

	// OrderInfo contains the information needed to create the p2c
	// script for an Orderbook contract.
	OrderInfo struct {
		SellerAccountID string   `json:"seller_account_id"`
		Prices          []*Price `json:"prices"`
	}

	// OpenOrder identifies a specific Orderbook contract.
	OpenOrder struct {
		bc.Outpoint
		bc.AssetAmount
		OrderInfo
		Script []byte
	}
)

// Misc. Orderbook errors.
var (
	ErrDuplicatePrice   = errors.New("attempt to create order with multiple prices using the same asset")
	ErrFractionalAmount = errors.New("attempt to buy a fractional amount of the offered asset")
	ErrMixedPayment     = errors.New("attempt to buy order with multiple assets as payment")
	ErrNoPayment        = errors.New("attempt to buy order with no payment")
	ErrNoPrices         = errors.New("attempt to create order with zero prices")
	ErrNonP2CScript     = errors.New("pkscript is not in p2c format")
	ErrNumParams        = errors.New("wrong number of parameters for orderbook contract")
	ErrOrderExceeded    = errors.New("attempt to buy more than is available from an order")
	ErrSameAsset        = errors.New("attempt to create order offering an asset in exchange for the same asset")
	ErrTooManyPrices    = errors.New("attempt to create order with too many prices")
	ErrWrongAsset       = errors.New("attempt to redeem wrong asset type from an order")
	ErrWrongPrice       = errors.New("payment does not match price")
)

// Maximum number of entries in an OrderInfo Prices list.
const MaxPrices = 1 // TODO(bobg): Support multiple prices per order.

func (info *OrderInfo) generateScript(ctx context.Context, sellerScript []byte) (pkScript, contract, contractHash []byte, err error) {
	var params [][]byte

	if sellerScript == nil {
		sellerScript, err = scriptFromAccountID(ctx, info.SellerAccountID)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	for _, price := range info.Prices {
		params = append(params, txscript.Int64ToScriptBytes(int64(price.OfferAmount)))
		params = append(params, txscript.Int64ToScriptBytes(int64(price.PaymentAmount)))
		params = append(params, price.AssetID[:])
	}
	params = append(params, sellerScript)

	contract, err = buildContract(len(info.Prices))
	if err != nil {
		return nil, nil, nil, err
	}
	contractHash = txscript.Hash160(contract)

	addr, err := txscript.NewAddressContractHash(contractHash, params)
	if err != nil {
		return nil, nil, nil, err
	}
	return addr.ScriptAddress(), contract, contractHash, nil
}

// SellerScript returns the contract parameter indicating where
// payments for the offered asset should be sent.
func (openOrder *OpenOrder) SellerScript() ([]byte, error) {
	return extractSellerScript(openOrder.Script)
}

func extractSellerScript(pkscript []byte) ([]byte, error) {
	isP2C, _, params := txscript.TestPayToContract(pkscript)
	if !isP2C {
		return nil, ErrNonP2CScript
	}
	l := len(params)
	if l < 4 {
		return nil, ErrNumParams
	}
	if l%3 != 1 {
		return nil, ErrNumParams
	}
	return params[l-1], nil
}

func scriptFromAccountID(ctx context.Context, accountID string) ([]byte, error) {
	addr, err := appdb.NewAddress(ctx, accountID, true)
	if err != nil {
		return nil, errors.Wrapf(err, "generating address, accountID %s", accountID)
	}

	redeemHash := hash160.Sum(addr.RedeemScript)

	sb := txscript.NewScriptBuilder()
	sb.AddOp(txscript.OP_DUP)
	sb.AddOp(txscript.OP_HASH160)
	sb.AddData(redeemHash[:])
	sb.AddOp(txscript.OP_EQUALVERIFY)
	sb.AddOp(txscript.OP_EVAL)

	return sb.Script()
}

// Build the contract script for an n-price orderbook order.
//
// TODO(bobg): When this is fleshed out with scripts for values of
// n>1, preserve the scripts in source form as comments and commit the
// actual script bytes here in the function.  That way we don't have
// to compute them anew each time, and we add a measure of defense
// against unexpected script (and therefore contract-hash) changes.
func buildContract(n int) ([]byte, error) {
	if n == 0 {
		return nil, ErrNoPrices
	}
	if n > MaxPrices {
		return nil, ErrTooManyPrices
	}

	// IMPORTANT! If you edit this script, you will change its contract
	// hash, and then any utxos containing the old contract hash will be
	// unredeemable.  So if you must edit it, be sure to preserve the
	// old version of the script as well somehow.
	const script1 = `
		4 ROLL
		IF
			5 PICK MUL
			4 PICK 2 PICK MUL
			ADD
			1 ROLL AMOUNT
			MUL
			EQUALVERIFY
			3 ROLL 1 ROLL 2 PICK
			REQUIREOUTPUT VERIFY
			ASSET
			SWAP
			DROP OUTPUTSCRIPT
			REQUIREOUTPUT
		ELSE
			DROP DROP DROP
			EVAL
		ENDIF
	`
	return txscript.ParseScriptString(script1)
}
