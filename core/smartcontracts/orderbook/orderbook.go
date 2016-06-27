package orderbook

import (
	"fmt"

	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/cos"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/log"
)

var scriptVersion = txscript.ScriptVersion1

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

		// SellerScript is only used by findOpenOrdersHelper to format API
		// responses.
		SellerScript chainjson.HexBytes `json:"script"`
	}

	// OpenOrder identifies a specific Orderbook contract.
	OpenOrder struct {
		bc.Outpoint
		bc.AssetAmount
		OrderInfo
		Script chainjson.HexBytes `json:"script"`
	}
)

// Misc. Orderbook errors.
var (
	ErrDuplicatePrice   = errors.New("attempt to create order with multiple prices using the same asset")
	ErrFractionalAmount = errors.New("attempt to buy a fractional amount of the offered asset")
	ErrMixedPayment     = errors.New("attempt to buy order with multiple assets as payment")
	ErrNoPayment        = errors.New("attempt to buy order with no payment")
	ErrNoPrices         = errors.New("attempt to create order with zero prices")
	ErrNumParams        = errors.New("wrong number of parameters for orderbook contract")
	ErrOrderExceeded    = errors.New("attempt to buy more than is available from an order")
	ErrSameAsset        = errors.New("attempt to create order offering an asset in exchange for the same asset")
	ErrTooManyPrices    = errors.New("attempt to create order with too many prices")
	ErrWrongAsset       = errors.New("attempt to redeem wrong asset type from an order")
	ErrWrongPrice       = errors.New("payment does not match price")
)

var (
	onePriceContract     []byte
	onePriceContractHash bc.ContractHash
)

// MaxPrices is the maximum number of entries in an OrderInfo Prices list.
const MaxPrices = 1 // TODO(bobg): Support multiple prices per order.

var fc *cos.FC

func Connect(chain *cos.FC) {
	if fc == chain {
		// Silently ignore duplicate calls.
		return
	}

	fc = chain

	fc.AddTxCallback(func(ctx context.Context, tx *bc.Tx) {
		// For outputs that match the orderbook p2c script format, index
		// orderbook-specific info in the db.
		for i, out := range tx.Outputs {
			isOrderbook, sellerScript, prices, err := testOrderbookScript(out.Script)
			if err != nil {
				log.Error(ctx, errors.Wrap(err, "testing for orderbook output script"))
				return
			}
			if isOrderbook {
				err = addOrderbookUTXO(ctx, tx, i, sellerScript, prices)
				if err != nil {
					log.Error(ctx, errors.Wrap(err, "adding orderbook utxo"))
					return
				}
			}
		}
	})
	fc.AddBlockCallback(addBlock)
}

// Note, FC guarantees it will call the tx callback
// for every tx in b before we get here.
func addBlock(ctx context.Context, b *bc.Block, conflicts []*bc.Tx) {
	deltxhash, delindex := prevoutDBKeys(b.Transactions...)
	const utxoDelQ = `
		DELETE FROM orderbook_utxos
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	_, err := pg.Exec(ctx, utxoDelQ, deltxhash, delindex)
	if err != nil {
		log.Write(ctx, "block", b.Height, "error", errors.Wrap(err))
		panic(err)
	}
	const priceDelQ = `
		DELETE FROM orderbook_prices
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	_, err = pg.Exec(ctx, priceDelQ, deltxhash, delindex)
	if err != nil {
		log.Write(ctx, "block", b.Height, "error", errors.Wrap(err))
		panic(err)
	}
}

func prevoutDBKeys(txs ...*bc.Tx) (txhash pg.Strings, index pg.Uint32s) {
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			txhash = append(txhash, in.Previous.Hash.String())
			index = append(index, in.Previous.Index)
		}
	}
	return
}

func (info *OrderInfo) generateScript(ctx context.Context, sellerScript []byte) (pkscript, contract []byte, err error) {
	if sellerScript == nil {
		sellerScript, err = scriptFromAccountID(ctx, info.SellerAccountID)
		if err != nil {
			return nil, nil, errors.Wrap(err, "getting account script")
		}
	}

	params := make([]txscript.Item, 0, 3*len(info.Prices)+1)

	for _, price := range info.Prices {
		params = append(params, txscript.NumItem(int64(price.OfferAmount)))
		params = append(params, txscript.NumItem(int64(price.PaymentAmount)))
		params = append(params, txscript.DataItem(price.AssetID[:]))
	}
	params = append(params, txscript.DataItem(sellerScript))

	contract, err = buildContract(len(info.Prices))
	if err != nil {
		return nil, nil, errors.Wrap(err, "building contract")
	}

	pkscript, err = txscript.PayToContractHash(sha3.Sum256(contract), params, scriptVersion)
	return pkscript, contract, errors.Wrap(err, "building pkscript")
}

// SellerScript returns the contract parameter indicating where
// payments for the offered asset should be sent.
func (openOrder *OpenOrder) SellerScript() ([]byte, error) {
	return extractSellerScript(openOrder.Script)
}

// testOrderbookScript tests whether the given pkscript is an
// orderbook contract.  Returns true, the seller script, and the list
// of prices if so; false, nil, and nil otherwise.
func testOrderbookScript(pkscript []byte) (isOrderbook bool, sellerScript []byte, prices []*Price, err error) {
	scriptVersion, _, _, params := txscript.ParseP2C(pkscript, onePriceContract)
	if scriptVersion == nil {
		return false, nil, nil, nil
	}
	if len(params) < 4 {
		return false, nil, nil, nil
	}
	if (len(params)-1)%3 != 0 {
		return false, nil, nil, nil
	}

	prices = make([]*Price, 0, (len(params)-1)/3)
	for i := 0; i < len(params)-1; i += 3 {
		offerAmount, err := txscript.MakeScriptNumWithMaxLen(params[i], false, len(params[i]))
		if err != nil {
			return false, nil, nil, errors.Wrapf(err, "offerAmount %v", params[i])
		}
		paymentAmount, err := txscript.MakeScriptNumWithMaxLen(params[i+1], false, len(params[i+1]))
		if err != nil {
			return false, nil, nil, errors.Wrapf(err, "paymentAmount %v", params[i+1])
		}
		assetID := params[i+2]
		price := &Price{
			OfferAmount:   uint64(offerAmount),
			PaymentAmount: uint64(paymentAmount),
		}
		copy(price.AssetID[:], assetID)
		prices = append(prices, price)
	}
	return true, params[len(params)-1], prices, nil
}

func extractSellerScript(pkscript []byte) ([]byte, error) {
	isOrderbook, sellerScript, _, err := testOrderbookScript(pkscript)
	if !isOrderbook {
		pkscriptStr, _ := txscript.DisasmString(pkscript)
		return nil, fmt.Errorf("extractSellerScript called on non-orderbook script [%s]", pkscriptStr)
	}
	return sellerScript, err
}

func scriptFromAccountID(ctx context.Context, accountID string) ([]byte, error) {
	addr, err := appdb.NewAddress(ctx, accountID, true)
	if err != nil {
		return nil, errors.Wrapf(err, "generating address, accountID %s", accountID)
	}

	return addr.PKScript, nil
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
			RESERVEOUTPUT VERIFY
			ASSET
			SWAP
			DROP OUTPUTSCRIPT
			RESERVEOUTPUT
		ELSE
			DROP DROP DROP
			EVAL
		ENDIF
	`
	return txscript.ParseScriptString(script1)
}

func init() {
	onePriceContract, _ = buildContract(1)
	onePriceContractHash = sha3.Sum256(onePriceContract)
}
