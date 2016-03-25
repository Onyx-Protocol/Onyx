package orderbook

import (
	"fmt"
	"testing"

	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/issuer"
	"chain/api/txbuilder"
	"chain/cos/bc"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestFindOpenOrders(t *testing.T) {
	ctx := pgtest.NewContext(t)

	fc, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}
	Connect(fc)

	projectID := assettest.CreateProjectFixture(ctx, t, "", "")
	managerNodeID := assettest.CreateManagerNodeFixture(ctx, t, projectID, "", nil, nil)
	issuerNodeID := assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
	accountID := assettest.CreateAccountFixture(ctx, t, managerNodeID, "", nil)
	assetID1 := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")
	assetID2 := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")
	assetID3 := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")

	openOrders, err := FindOpenOrders(ctx, []bc.AssetID{assetID1}, []bc.AssetID{})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	testutil.ExpectEqual(t, len(openOrders), 0, "expected no results from FindOpenOrders")

	prices := []*Price{
		&Price{
			AssetID:       assetID2,
			OfferAmount:   1,
			PaymentAmount: 1,
		},
	}

	asset1x100 := &bc.AssetAmount{
		AssetID: assetID1,
		Amount:  100,
	}

	issueDest, err := asset.NewAccountDestination(ctx, asset1x100, accountID, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	txTemplate, err := issuer.Issue(ctx, assetID1, []*txbuilder.Destination{issueDest})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	_, err = asset.FinalizeTx(ctx, txTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	offerTxTemplate, err := offer(ctx, accountID, asset1x100, prices, ttl)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	assettest.SignTxTemplate(t, offerTxTemplate, testutil.TestXPrv)

	_, err = asset.FinalizeTx(ctx, offerTxTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	openOrders, err = FindOpenOrders(ctx, []bc.AssetID{assetID2}, []bc.AssetID{})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	testutil.ExpectEqual(t, len(openOrders), 0, "expected no results from FindOpenOrders(assetID2, {}) [1]")

	openOrders, err = FindOpenOrders(ctx, []bc.AssetID{assetID1}, []bc.AssetID{assetID3})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	testutil.ExpectEqual(t, len(openOrders), 0, "expected no results from FindOpenOrders(assetID1, {assetID3})")

	combinations := []struct {
		offeredAssetIDs, paymentAssetIDs []bc.AssetID
	}{
		{[]bc.AssetID{assetID1}, []bc.AssetID{assetID2}},
		{[]bc.AssetID{assetID1}, nil},
		{nil, []bc.AssetID{assetID2}},
		{[]bc.AssetID{assetID1, assetID3}, []bc.AssetID{assetID2, assetID3}},
		{[]bc.AssetID{assetID1, assetID3}, nil},
		{nil, []bc.AssetID{assetID2, assetID3}},
	}
	for i, combination := range combinations {
		openOrders, err = FindOpenOrders(ctx, combination.offeredAssetIDs, combination.paymentAssetIDs)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		testutil.ExpectEqual(t, len(openOrders), 1, fmt.Sprintf("expected 1 result from FindOpenOrders (case %d)", i))
		openOrder := openOrders[0]
		testutil.ExpectEqual(t, openOrder.AssetID, assetID1, fmt.Sprintf("wrong assetID in result of FindOpenOrders (case %d)", i))
		testutil.ExpectEqual(t, openOrder.OrderInfo.SellerAccountID, accountID, fmt.Sprintf("wrong accountID in result of FindOpenOrders (case %d)", i))
		testutil.ExpectEqual(t, openOrder.Amount, uint64(100), fmt.Sprintf("wrong amount in result of FindOpenOrders (case %d)", i))
		testutil.ExpectEqual(t, openOrder.OrderInfo.Prices, prices, fmt.Sprintf("wrong prices in result of FindOpenOrders (case %d)", i))
	}

	openOrders, err = FindOpenOrders(ctx, nil, []bc.AssetID{assetID1})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	testutil.ExpectEqual(t, len(openOrders), 0, "expected no results from FindOpenOrders({}, {assetID1})")

	asset3x100 := &bc.AssetAmount{
		AssetID: assetID3,
		Amount:  100,
	}

	issueDest, err = asset.NewAccountDestination(ctx, asset3x100, accountID, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	txTemplate, err = issuer.Issue(ctx, assetID3, []*txbuilder.Destination{issueDest})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	_, err = asset.FinalizeTx(ctx, txTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	offerTxTemplate, err = offer(ctx, accountID, asset3x100, prices, ttl)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	assettest.SignTxTemplate(t, offerTxTemplate, testutil.TestXPrv)

	_, err = asset.FinalizeTx(ctx, offerTxTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	openOrders, err = FindOpenOrders(ctx, []bc.AssetID{assetID2}, []bc.AssetID{})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	testutil.ExpectEqual(t, len(openOrders), 0, "expected no results from FindOpenOrders(assetID2, {}) [2]")

	openOrders, err = FindOpenOrders(ctx, []bc.AssetID{assetID3}, []bc.AssetID{})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	testutil.ExpectEqual(t, len(openOrders), 1, "expected 1 result from FindOpenOrders(assetID3, {})")
	openOrder := openOrders[0]
	testutil.ExpectEqual(t, openOrder.AssetID, assetID3, "wrong assetID in result of FindOpenOrders(assetID3, {})")
	testutil.ExpectEqual(t, openOrder.OrderInfo.SellerAccountID, accountID, "wrong accountID in result of FindOpenOrders(assetID3, {})")
	testutil.ExpectEqual(t, openOrder.Amount, uint64(100), "wrong amount in result of FindOpenOrders(assetID3, {})")
	testutil.ExpectEqual(t, openOrder.OrderInfo.Prices, prices, "wrong prices in result of FindOpenOrders(assetID3, {})")
}
