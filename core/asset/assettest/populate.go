package assettest

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"testing"

	"golang.org/x/net/context"

	"chain/core/asset"
	"chain/core/txbuilder"
	"chain/cos/bc"
)

// PopulateCallbacks is a container for different callbacks used during Populate.
type PopulateCallbacks struct {
	// Called after an asset is created
	Asset func(bc.AssetID)

	// Called after an account is created
	Account func(string)

	// Called after some amount of some asset is issued to some account
	Issue func(bc.AssetID, string, uint64)

	// Called after a tx is submitted that trades "share" units of
	// shareAssetID from sellerID to buyerID in exchange for "dollars"
	// units of "usdAssetID"
	Trade func(sellerID, buyerID string, shareAssetID, usdAssetID bc.AssetID, shares, dollars uint64)

	// Called after all issuances are done and before any trades are
	// done.  Argument is the number of trades that is about to happen.
	AfterIssue func(nTrades int)
}

func (cb *PopulateCallbacks) doAsset(assetID bc.AssetID) {
	if cb != nil && cb.Asset != nil {
		cb.Asset(assetID)
	}
}

func (cb *PopulateCallbacks) doAccount(accountID string) {
	if cb != nil && cb.Account != nil {
		cb.Account(accountID)
	}
}

func (cb *PopulateCallbacks) doIssue(assetID bc.AssetID, accountID string, amount uint64) {
	if cb != nil && cb.Issue != nil {
		cb.Issue(assetID, accountID, amount)
	}
}

func (cb *PopulateCallbacks) doTrade(sellerID, buyerID string, shareAssetID, usdAssetID bc.AssetID, shares, dollars uint64) {
	if cb != nil && cb.Trade != nil {
		cb.Trade(sellerID, buyerID, shareAssetID, usdAssetID, shares, dollars)
	}
}

func (cb *PopulateCallbacks) doAfterIssue(nTrades int) {
	if cb != nil && cb.AfterIssue != nil {
		cb.AfterIssue(nTrades)
	}
}

// Populate reads a collection of trades from a file and executes
// them, first creating assets and accounts as needed and issuing
// necessary amounts of funds.
// The input is a CSV file whose lines have the format:
//   price,shares,buyer,seller,stockID
// (Format courtesy of the Glitterco testsuite.)  Price is an integer
// number of (notional) dollars that "buyer" pays to "seller" in
// exchange for "shares" units of "stockID."
// The filename parameter is the name of a CSV file relative to
// $CHAIN/core/asset/assettest/testdata.
// Various callbacks can be specified via cb.
func Populate(ctx context.Context, tb testing.TB, filename string, cb *PopulateCallbacks) {
	fullpath := os.Getenv("CHAIN") + "/core/asset/assettest/testdata/" + filename
	f, err := os.Open(fullpath)
	if err != nil {
		tb.Fatal(err)
	}
	defer f.Close()

	type (
		accountAssetPair struct {
			accountID string
			assetID   bc.AssetID
		}
		trade struct {
			dollars, shares             uint64
			shareAssetID                bc.AssetID
			shareSellerID, shareBuyerID string
		}
		balanceMap map[accountAssetPair]int64
	)

	inodeID := CreateIssuerNodeFixture(ctx, tb, "", "", nil, nil)
	mnodeID := CreateManagerNodeFixture(ctx, tb, "", "", nil, nil)

	// We call them dollars but it could be anything.
	usdAssetID := CreateAssetFixture(ctx, tb, inodeID, "", "")

	// Maps input "buyer" and "seller" ids to Chain accountIDs.
	accountMap := make(map[string]string)

	// Maps input "stockIDs" to Chain assetIDs.
	assetMap := make(map[string]bc.AssetID)

	// Records the lowest balance seen for each account/asset pair
	// during a prescan of the trades in the input.
	lowWaterMarks := make(map[accountAssetPair]int64)

	balances := make(balanceMap)

	var trades []*trade

	csvReader := csv.NewReader(f)
	for {
		// This loop reads the trades that are CSV-encoded in the testdata
		// file.  It records them in the "trades" slice, creating assets
		// and accounts as needed, and tracking balances as it goes.  In
		// tracking balances, we keep track of the low-water mark (the
		// minimum value seen) for each asset/account pair.  After this
		// loop, any low-water marks that are below zero indicate
		// issuances that must happen before any trades, to ensure no
		// trade will encounter insufficient-funds errors.

		rawitem, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tb.Fatal(err)
		}

		rawPrice := mustParseInt(rawitem[0])
		rawShares := mustParseInt(rawitem[1])
		rawBuyer := rawitem[2]
		rawSeller := rawitem[3]
		rawStockID := rawitem[4]

		if rawBuyer == rawSeller {
			continue
		}

		var (
			assetID bc.AssetID
			ok      bool
		)
		if assetID, ok = assetMap[rawStockID]; !ok {
			assetID = CreateAssetFixture(ctx, tb, inodeID, "", "")
			assetMap[rawStockID] = assetID
			cb.doAsset(assetID)
		}

		var sellerAccountID string
		if sellerAccountID, ok = accountMap[rawSeller]; !ok {
			sellerAccountID = CreateAccountFixture(ctx, tb, mnodeID, rawSeller, nil)
			accountMap[rawSeller] = sellerAccountID
			cb.doAccount(sellerAccountID)
		}
		sellerAssetPair := accountAssetPair{sellerAccountID, assetID}
		balances[accountAssetPair{sellerAccountID, usdAssetID}] += rawPrice
		balances[sellerAssetPair] -= rawShares
		if balances[sellerAssetPair] < lowWaterMarks[sellerAssetPair] {
			lowWaterMarks[sellerAssetPair] = balances[sellerAssetPair]
		}

		var buyerAccountID string
		if buyerAccountID, ok = accountMap[rawBuyer]; !ok {
			buyerAccountID = CreateAccountFixture(ctx, tb, mnodeID, rawBuyer, nil)
			accountMap[rawBuyer] = buyerAccountID
			cb.doAccount(buyerAccountID)
		}
		buyerUSDassetPair := accountAssetPair{buyerAccountID, usdAssetID}
		balances[buyerUSDassetPair] -= rawPrice
		balances[accountAssetPair{buyerAccountID, assetID}] += rawShares
		if balances[buyerUSDassetPair] < lowWaterMarks[buyerUSDassetPair] {
			lowWaterMarks[buyerUSDassetPair] = balances[buyerUSDassetPair]
		}

		trades = append(trades, &trade{
			dollars:       uint64(rawPrice),
			shares:        uint64(rawShares),
			shareAssetID:  assetID,
			shareSellerID: sellerAccountID,
			shareBuyerID:  buyerAccountID,
		})
	}

	// Do the issuances indicated by lowWaterMarks
	for accountAsset, lowWaterMark := range lowWaterMarks {
		if lowWaterMark >= 0 {
			continue
		}
		amount := uint64(-lowWaterMark)
		IssueAssetsFixture(ctx, tb, accountAsset.assetID, amount, accountAsset.accountID)
		cb.doIssue(accountAsset.assetID, accountAsset.accountID, amount)
	}

	cb.doAfterIssue(len(trades))

	// Execute the trades
	for _, trade := range trades {
		s := []*txbuilder.Source{
			asset.NewAccountSource(ctx, &bc.AssetAmount{AssetID: trade.shareAssetID, Amount: trade.shares}, trade.shareSellerID, nil, nil, nil),
			asset.NewAccountSource(ctx, &bc.AssetAmount{AssetID: usdAssetID, Amount: trade.dollars}, trade.shareBuyerID, nil, nil, nil),
		}
		d := []*txbuilder.Destination{
			AccountDest(ctx, tb, trade.shareSellerID, usdAssetID, trade.dollars),
			AccountDest(ctx, tb, trade.shareBuyerID, trade.shareAssetID, trade.shares),
		}
		Transfer(ctx, tb, s, d)
		cb.doTrade(trade.shareSellerID, trade.shareBuyerID, trade.shareAssetID, usdAssetID, trade.shares, trade.dollars)
	}
}

func mustParseInt(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return n
}
