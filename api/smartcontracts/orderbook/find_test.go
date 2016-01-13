package orderbook

import (
	"testing"

	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/database/pg/pgtest"
	"chain/fedchain/bc"
	chaintest "chain/testutil"
)

func TestFindOpenOrders(t *testing.T) {
	ctx := assettest.NewContextWithGenesisBlock(t)
	defer pgtest.Finish(ctx)

	projectID := assettest.CreateProjectFixture(ctx, t, "", "")
	managerNodeID := assettest.CreateManagerNodeFixture(ctx, t, projectID, "", nil, nil)
	issuerNodeID := assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
	accountID := assettest.CreateAccountFixture(ctx, t, managerNodeID, "", nil)
	assetID1 := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "")
	assetID2 := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "")
	assetID3 := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "")

	openOrderChan, err := FindOpenOrders(ctx, assetID1, []bc.AssetID{})
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	openOrders := slurpOpenOrders(openOrderChan)
	chaintest.ExpectEqual(t, len(openOrders), 0, "expected no results from FindOpenOrders")

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

	issueDest, err := asset.NewAccountDestination(ctx, asset1x100, accountID, false, nil)
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	txTemplate, err := asset.Issue(ctx, assetID1.String(), []*asset.Destination{issueDest})
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	_, err = asset.FinalizeTx(ctx, txTemplate)
	if err != nil {
		chaintest.FatalErr(t, err)
	}

	offerTxTemplate, err := offer(ctx, accountID, asset1x100, prices, ttl)
	if err != nil {
		chaintest.FatalErr(t, err)
	}

	err = asset.SignTxTemplate(offerTxTemplate, chaintest.TestXPrv)
	if err != nil {
		chaintest.FatalErr(t, err)
	}

	_, err = asset.FinalizeTx(ctx, offerTxTemplate)
	if err != nil {
		chaintest.FatalErr(t, err)
	}

	openOrderChan, err = FindOpenOrders(ctx, assetID2, []bc.AssetID{})
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	openOrders = slurpOpenOrders(openOrderChan)
	chaintest.ExpectEqual(t, len(openOrders), 0, "expected no results from FindOpenOrders(assetID2, {}) [1]")

	openOrderChan, err = FindOpenOrders(ctx, assetID1, []bc.AssetID{assetID3})
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	openOrders = slurpOpenOrders(openOrderChan)
	chaintest.ExpectEqual(t, len(openOrders), 0, "expected no results from FindOpenOrders(assetID1, {assetID3})")

	openOrderChan, err = FindOpenOrders(ctx, assetID1, []bc.AssetID{assetID2})
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	openOrders = slurpOpenOrders(openOrderChan)
	chaintest.ExpectEqual(t, len(openOrders), 1, "expected 1 result from FindOpenOrders(assetID1, {assetID2}) [1]")
	openOrder := openOrders[0]
	chaintest.ExpectEqual(t, openOrder.AssetID, assetID1, "wrong assetID in result of FindOpenOrders(assetID1, {assetID2}) [1]")
	chaintest.ExpectEqual(t, openOrder.OrderInfo.SellerAccountID, accountID, "wrong accountID in result of FindOpenOrders(assetID1, {assetID2}) [1]")
	chaintest.ExpectEqual(t, openOrder.Amount, uint64(100), "wrong amount in result of FindOpenOrders(assetID1, {assetID2}) [1]")
	chaintest.ExpectEqual(t, openOrder.OrderInfo.Prices, prices, "wrong prices in result of FindOpenOrders(assetID1, {assetID2}) [1]")

	asset3x100 := &bc.AssetAmount{
		AssetID: assetID3,
		Amount:  100,
	}

	issueDest, err = asset.NewAccountDestination(ctx, asset3x100, accountID, false, nil)
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	txTemplate, err = asset.Issue(ctx, assetID3.String(), []*asset.Destination{issueDest})
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	_, err = asset.FinalizeTx(ctx, txTemplate)
	if err != nil {
		chaintest.FatalErr(t, err)
	}

	offerTxTemplate, err = offer(ctx, accountID, asset3x100, prices, ttl)
	if err != nil {
		chaintest.FatalErr(t, err)
	}

	err = asset.SignTxTemplate(offerTxTemplate, chaintest.TestXPrv)
	if err != nil {
		chaintest.FatalErr(t, err)
	}

	_, err = asset.FinalizeTx(ctx, offerTxTemplate)
	if err != nil {
		chaintest.FatalErr(t, err)
	}

	openOrderChan, err = FindOpenOrders(ctx, assetID2, []bc.AssetID{})
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	openOrders = slurpOpenOrders(openOrderChan)
	chaintest.ExpectEqual(t, len(openOrders), 0, "expected no results from FindOpenOrders(assetID2, {}) [2]")

	openOrderChan, err = FindOpenOrders(ctx, assetID1, []bc.AssetID{assetID2})
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	openOrders = slurpOpenOrders(openOrderChan)
	chaintest.ExpectEqual(t, len(openOrders), 1, "expected 1 result from FindOpenOrders(assetID1, {assetID2}) [2]")
	openOrder = openOrders[0]
	chaintest.ExpectEqual(t, openOrder.AssetID, assetID1, "wrong assetID in result of FindOpenOrders(assetID1, {assetID2}) [2]")
	chaintest.ExpectEqual(t, openOrder.OrderInfo.SellerAccountID, accountID, "wrong accountID in result of FindOpenOrders(assetID1, {assetID2}) [2]")
	chaintest.ExpectEqual(t, openOrder.Amount, uint64(100), "wrong amount in result of FindOpenOrders(assetID1, {assetID2}) [2]")
	chaintest.ExpectEqual(t, openOrder.OrderInfo.Prices, prices, "wrong prices in result of FindOpenOrders(assetID1, {assetID2}) [2]")

	openOrderChan, err = FindOpenOrders(ctx, assetID3, []bc.AssetID{})
	if err != nil {
		chaintest.FatalErr(t, err)
	}
	openOrders = slurpOpenOrders(openOrderChan)
	chaintest.ExpectEqual(t, len(openOrders), 1, "expected 1 result from FindOpenOrders(assetID3, {})")
	openOrder = openOrders[0]
	chaintest.ExpectEqual(t, openOrder.AssetID, assetID3, "wrong assetID in result of FindOpenOrders(assetID3, {})")
	chaintest.ExpectEqual(t, openOrder.OrderInfo.SellerAccountID, accountID, "wrong accountID in result of FindOpenOrders(assetID3, {})")
	chaintest.ExpectEqual(t, openOrder.Amount, uint64(100), "wrong amount in result of FindOpenOrders(assetID3, {})")
	chaintest.ExpectEqual(t, openOrder.OrderInfo.Prices, prices, "wrong prices in result of FindOpenOrders(assetID3, {})")
}
