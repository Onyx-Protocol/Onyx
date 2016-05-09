package api

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/generator"
	"chain/api/issuer"
	"chain/api/smartcontracts/orderbook"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/net/http/httpjson"
	"chain/testutil"
)

type contractsFixtureInfo struct {
	projectID, managerNodeID, issuerNodeID, sellerAccountID string
	aaplAssetID, usdAssetID                                 bc.AssetID
	prices                                                  []*orderbook.Price
}

var ttl = time.Hour

func TestOfferContract(t *testing.T) {
	withContractsFixture(t, func(ctx context.Context, fixtureInfo *contractsFixtureInfo) {
		buildRequest := &BuildRequest{
			Sources: []*Source{
				&Source{
					AssetID:   &fixtureInfo.aaplAssetID,
					Amount:    100,
					AccountID: fixtureInfo.sellerAccountID,
					Type:      "account",
				},
			},
			Dests: []*Destination{
				&Destination{
					AssetID:   &fixtureInfo.aaplAssetID,
					Amount:    100,
					AccountID: fixtureInfo.sellerAccountID,
					OrderbookPrices: []*orderbook.Price{
						&orderbook.Price{
							AssetID:       fixtureInfo.usdAssetID,
							OfferAmount:   1,
							PaymentAmount: 110,
						},
					},
					Type: "orderbook",
				},
			},
		}
		callBuildSingle(t, ctx, buildRequest, func(txTemplate *txbuilder.Template) {
			assettest.SignTxTemplate(t, txTemplate, testutil.TestXPrv)

			offerTx, err := asset.FinalizeTx(ctx, txTemplate)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			if len(offerTx.Outputs) != 1 {
				t.Fatalf("got %d outputs, want %d", len(offerTx.Outputs), 1)
			}

			if offerTx.Outputs[0].AssetID != fixtureInfo.aaplAssetID {
				t.Fatalf("wrong asset id. got %s, want %s", offerTx.Outputs[0].AssetID, fixtureInfo.aaplAssetID)
			}

			if offerTx.Outputs[0].Amount != 100 {
				t.Fatalf("wrong amount. got %d, want %d", offerTx.Outputs[0].Amount, 100)
			}
		})
	})
}

func callBuildSingle(t *testing.T, ctx context.Context, request *BuildRequest, continuation func(*txbuilder.Template)) {
	result, err := buildSingle(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if dict, ok := result.(map[string]interface{}); ok {
		if template, ok := dict["template"]; ok {
			if txTemplate, ok := template.(*txbuilder.Template); ok {
				continuation(txTemplate)
			} else {
				t.Fatal("expected result[\"template\"] to be a TxTemplate")
			}
		} else {
			t.Fatal("expected result to contain \"template\"")
		}
	} else {
		t.Fatal("expected result to be a map")
	}
}

func TestFindAndBuyContract(t *testing.T) {
	withContractsFixture(t, func(ctx context.Context, fixtureInfo *contractsFixtureInfo) {
		openOrder, err := offerAndFind(ctx, t, fixtureInfo)
		if err != nil {
			t.Fatal(err)
		}

		buyerAccountID := assettest.CreateAccountFixture(ctx, t, fixtureInfo.managerNodeID, "buyer", nil)

		// Issue USD assets to buy with
		usd2200 := &bc.AssetAmount{
			AssetID: fixtureInfo.usdAssetID,
			Amount:  2200,
		}
		issueDest, err := asset.NewAccountDestination(ctx, usd2200, buyerAccountID, nil)
		if err != nil {
			t.Fatal(err)
		}
		issueTxTemplate, err := issuer.Issue(ctx, fixtureInfo.usdAssetID, []*txbuilder.Destination{issueDest})
		if err != nil {
			t.Fatal(err)
		}
		_, err = asset.FinalizeTx(ctx, issueTxTemplate)
		if err != nil {
			t.Fatal(err)
		}

		buildRequest := &BuildRequest{
			Sources: []*Source{
				&Source{
					AssetID:   &fixtureInfo.usdAssetID,
					Amount:    2200,
					AccountID: buyerAccountID,
					Type:      "account",
				},
				&Source{
					Amount:         20, // shares of AAPL
					PaymentAssetID: &fixtureInfo.usdAssetID,
					PaymentAmount:  2200, // USD
					TxHash:         &openOrder.Hash,
					Index:          &openOrder.Index,
					Type:           "orderbook-redeem",
				},
			},
			Dests: []*Destination{
				&Destination{
					AssetID: &fixtureInfo.usdAssetID,
					Amount:  2200,
					Address: openOrder.OrderInfo.SellerScript,
					Type:    "address",
				},
				&Destination{
					AssetID:   &fixtureInfo.aaplAssetID,
					Amount:    20,
					AccountID: buyerAccountID,
					Type:      "account",
				},
			},
		}
		callBuildSingle(t, ctx, buildRequest, func(txTemplate *txbuilder.Template) {
			assettest.SignTxTemplate(t, txTemplate, testutil.TestXPrv)

			buyTx, err := asset.FinalizeTx(ctx, txTemplate)
			if err != nil {
				t.Fatal(err)
			}

			assettest.ExpectMatchingOutputs(t, buyTx, 1, "sending payment to seller", func(t *testing.T, txOutput *bc.TxOutput) bool {
				return reflect.DeepEqual(txOutput.Script, []byte(openOrder.OrderInfo.SellerScript))
			})
		})
	})
}

func offerAndFind(ctx context.Context, t testing.TB, fixtureInfo *contractsFixtureInfo) (*orderbook.OpenOrder, error) {
	assetAmount := &bc.AssetAmount{
		AssetID: fixtureInfo.aaplAssetID,
		Amount:  100,
	}
	source := asset.NewAccountSource(ctx, assetAmount, fixtureInfo.sellerAccountID, nil, nil)
	sources := []*txbuilder.Source{source}

	orderInfo := &orderbook.OrderInfo{
		SellerAccountID: fixtureInfo.sellerAccountID,
		Prices:          fixtureInfo.prices,
	}

	destination, err := orderbook.NewDestination(ctx, assetAmount, orderInfo, nil)
	if err != nil {
		return nil, err
	}
	destinations := []*txbuilder.Destination{destination}

	offerTxTemplate, err := txbuilder.Build(ctx, nil, sources, destinations, nil, ttl)
	if err != nil {
		return nil, err
	}
	assettest.SignTxTemplate(t, offerTxTemplate, testutil.TestXPrv)
	_, err = asset.FinalizeTx(ctx, offerTxTemplate)
	if err != nil {
		return nil, err
	}

	req1 := globalFindOrder{
		OfferedAssetIDs: []bc.AssetID{fixtureInfo.aaplAssetID},
		PaymentAssetIDs: []bc.AssetID{fixtureInfo.usdAssetID},
	}

	// Need to add an http request to the context before running Find
	httpURL, err := url.Parse("http://boop.bop/v3/contracts/orderbook?status=open")
	httpReq := http.Request{URL: httpURL}
	ctx = httpjson.WithRequest(ctx, &httpReq)

	// Now find that open order
	openOrders, err := findOrders(ctx, req1)
	if err != nil {
		return nil, err
	}

	if len(openOrders) != 1 {
		return nil, fmt.Errorf("expected 1 open order, got %d", len(openOrders))
	}

	return openOrders[0], nil
}

func TestFindAndCancelContract(t *testing.T) {
	withContractsFixture(t, func(ctx context.Context, fixtureInfo *contractsFixtureInfo) {
		openOrder, err := offerAndFind(ctx, t, fixtureInfo)
		if err != nil {
			t.Fatal(err)
		}
		buildRequest := &BuildRequest{
			Sources: []*Source{
				&Source{
					TxHash: &openOrder.Hash,
					Index:  &openOrder.Index,
					Type:   "orderbook-cancel",
				},
			},
			Dests: []*Destination{
				&Destination{
					AssetID:   &fixtureInfo.aaplAssetID,
					Amount:    100,
					AccountID: fixtureInfo.sellerAccountID,
					Type:      "account",
				},
			},
		}
		callBuildSingle(t, ctx, buildRequest, func(txTemplate *txbuilder.Template) {
			assettest.SignTxTemplate(t, txTemplate, testutil.TestXPrv)

			_, err := asset.FinalizeTx(ctx, txTemplate)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			// Make a block so the order should go away
			_, err = generator.MakeBlock(ctx)
			if err != nil {
				t.Fatal(err)
			}

			// Make sure that order is gone now
			found, err := orderbook.FindOpenOrderByOutpoint(ctx, &openOrder.Outpoint)
			if err != nil {
				t.Fatal(err)
			}
			if found != nil {
				t.Fatal("expected order to be gone after cancellation and block-landing")
			}
		})
	})
}

func TestFindBySeller(t *testing.T) {
	withContractsFixture(t, func(ctx context.Context, fixtureInfo *contractsFixtureInfo) {
		order, err := offerAndFind(ctx, t, fixtureInfo)
		if err != nil {
			t.Fatal(err)
		}

		expectedOrders := []*orderbook.OpenOrder{order}

		foundOrders, err := callFindAccountOrders(ctx, fixtureInfo.sellerAccountID)
		if err != nil {
			t.Fatal(err)
		}
		testutil.ExpectEqual(t, foundOrders, expectedOrders, "find by seller [1]")

		foundOrders, err = callFindAccountOrders(ctx, fixtureInfo.sellerAccountID+"x")
		if err != nil {
			t.Fatal(err)
		}
		if foundOrders != nil {
			t.Errorf("find by seller [2]: got %v, expected nil", foundOrders)
		}
	})
}

func callFindAccountOrders(ctx context.Context, accountID string) ([]*orderbook.OpenOrder, error) {
	// Need to add an http request to the context before running Find
	httpURL, err := url.Parse("http://boop.bop/v3/contracts/orderbook?status=open")
	if err != nil {
		return nil, err
	}
	httpReq := http.Request{URL: httpURL}
	ctx = httpjson.WithRequest(ctx, &httpReq)
	return findAccountOrders(ctx, accountID)
}

func withContractsFixture(t *testing.T, fn func(context.Context, *contractsFixtureInfo)) {
	ctx := pgtest.NewContext(t)
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB))
	fc, err := assettest.InitializeSigningGenerator(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	orderbook.Connect(fc)

	var fixtureInfo contractsFixtureInfo

	fixtureInfo.projectID = assettest.CreateProjectFixture(ctx, t, "", "")
	fixtureInfo.managerNodeID = assettest.CreateManagerNodeFixture(ctx, t, fixtureInfo.projectID, "", nil, nil)
	fixtureInfo.issuerNodeID = assettest.CreateIssuerNodeFixture(ctx, t, fixtureInfo.projectID, "", nil, nil)
	fixtureInfo.sellerAccountID = assettest.CreateAccountFixture(ctx, t, fixtureInfo.managerNodeID, "seller", nil)
	fixtureInfo.aaplAssetID = assettest.CreateAssetFixture(ctx, t, fixtureInfo.issuerNodeID, "", "")
	fixtureInfo.usdAssetID = assettest.CreateAssetFixture(ctx, t, fixtureInfo.issuerNodeID, "", "")
	fixtureInfo.prices = []*orderbook.Price{
		&orderbook.Price{
			AssetID:       fixtureInfo.usdAssetID,
			OfferAmount:   1,
			PaymentAmount: 110,
		},
	}

	aapl100 := &bc.AssetAmount{
		AssetID: fixtureInfo.aaplAssetID,
		Amount:  100,
	}
	issueDest, err := asset.NewAccountDestination(ctx, aapl100, fixtureInfo.sellerAccountID, nil)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	txTemplate, err := issuer.Issue(ctx, fixtureInfo.aaplAssetID, []*txbuilder.Destination{issueDest})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	_, err = asset.FinalizeTx(ctx, txTemplate)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	fn(ctx, &fixtureInfo)
}
