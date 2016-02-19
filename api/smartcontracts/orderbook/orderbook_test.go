package orderbook

import (
	"os"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/generator"
	"chain/api/issuer"
	"chain/api/txbuilder"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
	"chain/testutil"
)

type orderbookFixtureInfo struct {
	projectID, managerNodeID, issuerNodeID, sellerAccountID string
	aaplAssetID, usdAssetID                                 bc.AssetID
	offerTx                                                 *bc.Tx
	openOrder                                               *OpenOrder
}

var ttl = time.Hour

func TestOffer(t *testing.T) {
	withOrderbookFixture(t, func(ctx context.Context, fixtureInfo *orderbookFixtureInfo) {
		numOutputs := len(fixtureInfo.offerTx.Outputs)
		testutil.ExpectEqual(t, numOutputs, 1, "wrong number of outputs")

		txOutput := fixtureInfo.offerTx.Outputs[0]
		testutil.ExpectEqual(t, txOutput.AssetID, fixtureInfo.aaplAssetID, "wrong asset id")
		testutil.ExpectEqual(t, txOutput.Amount, uint64(100), "wrong amount")
		expectPaysToOrderbookContract(ctx, t, fixtureInfo.openOrder, txOutput.Script, "does not pay to contract")
	})
}

func TestBuy(t *testing.T) {
	withOrderbookFixture(t, func(ctx context.Context, fixtureInfo *orderbookFixtureInfo) {
		buyerAccountID := assettest.CreateAccountFixture(ctx, t, fixtureInfo.managerNodeID, "buyer", nil)

		usd2200 := &bc.AssetAmount{
			AssetID: fixtureInfo.usdAssetID,
			Amount:  2200,
		}
		issueDest, err := asset.NewAccountDestination(ctx, usd2200, buyerAccountID, nil)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		issueTxTemplate, err := issuer.Issue(ctx, fixtureInfo.usdAssetID.String(), []*txbuilder.Destination{issueDest})
		if err != nil {
			testutil.FatalErr(t, err)
		}
		_, err = asset.FinalizeTx(ctx, issueTxTemplate)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		funds := asset.NewAccountSource(ctx, usd2200, buyerAccountID)

		aapl20 := &bc.AssetAmount{
			AssetID: fixtureInfo.aaplAssetID,
			Amount:  20,
		}
		buyerDest, err := asset.NewAccountDestination(ctx, aapl20, buyerAccountID, nil)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		buyTxTemplate, err := buy(ctx, fixtureInfo.openOrder, funds, buyerDest, ttl)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		assettest.SignTxTemplate(t, buyTxTemplate, testutil.TestXPrv)

		buyTx, err := asset.FinalizeTx(ctx, buyTxTemplate)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		testutil.ExpectEqual(t, len(buyTx.Inputs), 2, "wrong number of buyTx inputs")

		assettest.ExpectMatchingInputs(t, buyTx, 1, "redeeming p2c asset", func(t *testing.T, txInput *bc.TxInput) bool {
			if !reflect.DeepEqual(fixtureInfo.openOrder.Outpoint, txInput.Previous) {
				return false
			}
			if !redeemsOrderbookContract(ctx, &fixtureInfo.openOrder.OrderInfo, txInput.SignatureScript) {
				return false
			}
			return true
		})

		testutil.ExpectEqual(t, len(buyTx.Outputs), 3, "wrong number of buyTx outputs")

		assettest.ExpectMatchingOutputs(t, buyTx, 1, "sending p2c asset to buyer", func(t *testing.T, txOutput *bc.TxOutput) bool {
			if !reflect.DeepEqual(txOutput.AssetID, fixtureInfo.aaplAssetID) {
				return false
			}
			if txOutput.Amount != 20 {
				return false
			}
			if !paysToAccount(ctx, t, buyerAccountID, txOutput.Script) {
				return false
			}
			return true
		})
		assettest.ExpectMatchingOutputs(t, buyTx, 1, "sending p2c payment to seller", func(t *testing.T, txOutput *bc.TxOutput) bool {
			if !reflect.DeepEqual(txOutput.AssetID, fixtureInfo.usdAssetID) {
				return false
			}
			if txOutput.Amount != 2200 {
				return false
			}
			sellerScript, err := fixtureInfo.openOrder.SellerScript()
			if err != nil {
				return false
			}
			if !paysToScript(ctx, txOutput.Script, sellerScript) {
				return false
			}
			return true
		})
		assettest.ExpectMatchingOutputs(t, buyTx, 1, "sending p2c change to contract", func(t *testing.T, txOutput *bc.TxOutput) bool {
			if !reflect.DeepEqual(txOutput.AssetID, fixtureInfo.aaplAssetID) {
				return false
			}
			if txOutput.Amount != 80 {
				return false
			}
			if !reflect.DeepEqual(txOutput.Script, []byte(fixtureInfo.openOrder.Script)) {
				return false
			}
			return true
		})
	})
}

func TestCancel(t *testing.T) {
	withOrderbookFixture(t, func(ctx context.Context, fixtureInfo *orderbookFixtureInfo) {
		cancelTxTemplate, err := cancel(ctx, fixtureInfo.openOrder, ttl)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		assettest.SignTxTemplate(t, cancelTxTemplate, testutil.TestXPrv)
		cancelTx, err := asset.FinalizeTx(ctx, cancelTxTemplate)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		testutil.ExpectEqual(t, len(cancelTx.Inputs), 1, "wrong number of cancelTx inputs")
		testutil.ExpectEqual(t, cancelTx.Inputs[0].Previous, fixtureInfo.openOrder.Outpoint, "wrong cancelTx prevout")

		testutil.ExpectEqual(t, len(cancelTx.Outputs), 1, "wrong number of cancelTx outputs")
		output := cancelTx.Outputs[0]
		testutil.ExpectEqual(t, output.AssetID, fixtureInfo.aaplAssetID, "wrong cancelTx asset")
		testutil.ExpectEqual(t, output.Amount, uint64(100), "wrong cancelTx amount")

		expectPaysToAccount(ctx, t, fixtureInfo.sellerAccountID, output.Script)

		found, err := FindOpenOrders(ctx, []bc.AssetID{fixtureInfo.aaplAssetID}, nil)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		testutil.ExpectEqual(t, len(found), 0, "expected no cancelable orders [1]")

		_, err = generator.MakeBlock(ctx)
		if err != nil {
			t.Fatal(err)
		}

		found, err = FindOpenOrders(ctx, []bc.AssetID{fixtureInfo.aaplAssetID}, nil)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		testutil.ExpectEqual(t, len(found), 0, "expected no cancelable orders [2]")
	})
}

func withOrderbookFixture(t *testing.T, fn func(ctx context.Context, fixtureInfo *orderbookFixtureInfo)) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	fc, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	ConnectFedchain(fc)

	var fixtureInfo orderbookFixtureInfo

	fixtureInfo.projectID = assettest.CreateProjectFixture(ctx, t, "", "")
	fixtureInfo.managerNodeID = assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.projectID, "", nil, nil)
	fixtureInfo.issuerNodeID = assettest.CreateIssuerNodeFixture(ctx, t, fixtureInfo.projectID, "", nil, nil)
	fixtureInfo.sellerAccountID = assettest.CreateAccountFixture(ctx, t, fixtureInfo.managerNodeID, "seller", nil)
	fixtureInfo.aaplAssetID = assettest.CreateAssetFixture(ctx, t, fixtureInfo.issuerNodeID, "", "")
	fixtureInfo.usdAssetID = assettest.CreateAssetFixture(ctx, t, fixtureInfo.issuerNodeID, "", "")

	aapl100 := &bc.AssetAmount{
		AssetID: fixtureInfo.aaplAssetID,
		Amount:  100,
	}

	issueDest, err := asset.NewAccountDestination(ctx, aapl100, fixtureInfo.sellerAccountID, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	issueTxTemplate, err := issuer.Issue(ctx, fixtureInfo.aaplAssetID.String(), []*txbuilder.Destination{issueDest})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	_, err = asset.FinalizeTx(ctx, issueTxTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	prices := []*Price{
		&Price{
			AssetID:       fixtureInfo.usdAssetID,
			OfferAmount:   1,
			PaymentAmount: 110,
		},
	}

	offerTxTemplate, err := offer(ctx, fixtureInfo.sellerAccountID, aapl100, prices, ttl)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	assettest.SignTxTemplate(t, offerTxTemplate, testutil.TestXPrv)

	fixtureInfo.offerTx, err = asset.FinalizeTx(ctx, offerTxTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	fixtureInfo.openOrder = &OpenOrder{
		Outpoint: bc.Outpoint{
			Hash:  fixtureInfo.offerTx.Hash,
			Index: 0,
		},
		AssetAmount: *aapl100,
		OrderInfo: OrderInfo{
			SellerAccountID: fixtureInfo.sellerAccountID,
			Prices:          prices,
		},
		Script: fixtureInfo.offerTx.Outputs[0].Script,
	}

	fn(ctx, &fixtureInfo)
}

func offer(ctx context.Context, sellerAccountID string, assetAmount *bc.AssetAmount, prices []*Price, ttl time.Duration) (*txbuilder.Template, error) {
	source := asset.NewAccountSource(ctx, assetAmount, sellerAccountID)
	sources := []*txbuilder.Source{source}

	orderInfo := &OrderInfo{
		SellerAccountID: sellerAccountID,
		Prices:          prices,
	}

	destination, err := NewDestination(ctx, assetAmount, orderInfo, nil)
	if err != nil {
		return nil, err
	}
	destinations := []*txbuilder.Destination{destination}

	return txbuilder.Build(ctx, nil, sources, destinations, nil, ttl)
}

func buy(ctx context.Context, order *OpenOrder, funds *txbuilder.Source, destination *txbuilder.Destination, ttl time.Duration) (*txbuilder.Template, error) {
	redeemSource := NewRedeemSource(order, destination.Amount, &funds.AssetAmount)
	sources := []*txbuilder.Source{funds, redeemSource}

	destinations := make([]*txbuilder.Destination, 0, 3)
	destinations = append(destinations, destination)

	sellerScript, err := order.SellerScript()
	if err != nil {
		return nil, err
	}
	sellerDestination := txbuilder.NewScriptDestination(ctx, &funds.AssetAmount, sellerScript, nil)
	if err != nil {
		return nil, err
	}
	destinations = append(destinations, sellerDestination)

	return txbuilder.Build(ctx, nil, sources, destinations, nil, ttl)
}

func cancel(ctx context.Context, order *OpenOrder, ttl time.Duration) (*txbuilder.Template, error) {
	cancelSource := NewCancelSource(order)
	sources := []*txbuilder.Source{cancelSource}

	destination, err := asset.NewAccountDestination(ctx, &order.AssetAmount, order.SellerAccountID, nil)
	if err != nil {
		return nil, err
	}
	destinations := []*txbuilder.Destination{destination}

	return txbuilder.Build(ctx, nil, sources, destinations, nil, ttl)
}

func expectPaysToAccount(ctx context.Context, t *testing.T, accountID string, script []byte) {
	if !paysToAccount(ctx, t, accountID, script) {
		t.Errorf("expected script to pay to account %s: %x", accountID, script)
	}
}

func paysToAccount(ctx context.Context, t testing.TB, accountID string, script []byte) bool {
	// first check utxos
	const q = `
		SELECT account_id=$1 FROM utxos u
		JOIN account_utxos a ON (u.tx_hash, u.index) = (a.tx_hash, a.index)
		WHERE script=$2
	`
	var utxoMatch bool
	err := pg.FromContext(ctx).QueryRow(ctx, q, accountID, script).Scan(&utxoMatch)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if utxoMatch {
		return true
	}

	sellerScript, err := extractSellerScript(script)
	if err != nil {
		return false
	}
	addr, err := appdb.GetAddress(ctx, sellerScript)
	if errors.Root(err) == pg.ErrUserInputNotFound {
		return false
	} else if err != nil {
		testutil.FatalErr(t, err)
	}
	return addr.AccountID == accountID
}

func paysToScript(ctx context.Context, gotScript, expectedScript []byte) bool {
	return reflect.DeepEqual(gotScript, expectedScript)
}

func redeemsOrderbookContract(ctx context.Context, orderInfo *OrderInfo, script []byte) bool {
	_, contract, _, err := orderInfo.generateScript(ctx, nil)
	if err != nil {
		return false
	}
	pushedData, err := txscript.PushedData(script)
	if err != nil {
		return false
	}
	if len(pushedData) < 1 {
		return false
	}
	actualContract := pushedData[len(pushedData)-1]
	return reflect.DeepEqual(contract, actualContract)
}

func expectPaysToOrderbookContract(ctx context.Context, t *testing.T, openOrder *OpenOrder, script []byte, msg string) {
	sellerScript, err := openOrder.SellerScript()
	if err != nil {
		testutil.FatalErr(t, err)
	}
	expectedScript, _, _, err := openOrder.generateScript(ctx, sellerScript)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	testutil.ExpectScriptEqual(t, script, expectedScript, msg)
}

func slurpOpenOrders(ch <-chan *OpenOrder) (result []*OpenOrder) {
	for order := range ch {
		result = append(result, order)
	}
	return result
}

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	pgtest.Open(ctx, u, "orderbooktest", "../../appdb/schema.sql")
}
