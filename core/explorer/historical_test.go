package explorer

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/generator"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
)

func TestHistoricalOutputs(t *testing.T) {
	ctx := pgtest.NewContext(t)
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB))
	fc, err := assettest.InitializeSigningGenerator(ctx, store, nil)
	if err != nil {
		t.Fatal(err)
	}

	Connect(ctx, fc, true, 0, true)

	// TODO(bobg): The dataset-loading code here has broader
	// applicability for testing and benchmarking.  Migrate it to
	// assettest or someplace.

	f, err := os.Open("testdata/glittercosmall.csv")
	if err != nil {
		t.Fatal(err)
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

	inodeID := assettest.CreateIssuerNodeFixture(ctx, t, "", "", nil, nil)
	mnodeID := assettest.CreateManagerNodeFixture(ctx, t, "", "", nil, nil)

	usdAssetID := assettest.CreateAssetFixture(ctx, t, inodeID, "$", "")

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
			t.Fatal(err)
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
			assetID = assettest.CreateAssetFixture(ctx, t, inodeID, "", "")
			assetMap[rawStockID] = assetID
		}

		var sellerAccountID string
		if sellerAccountID, ok = accountMap[rawSeller]; !ok {
			sellerAccountID = assettest.CreateAccountFixture(ctx, t, mnodeID, rawSeller, nil)
			accountMap[rawSeller] = sellerAccountID
		}
		sellerAssetPair := accountAssetPair{sellerAccountID, assetID}
		balances[accountAssetPair{sellerAccountID, usdAssetID}] += rawPrice
		balances[sellerAssetPair] -= rawShares
		if balances[sellerAssetPair] < lowWaterMarks[sellerAssetPair] {
			lowWaterMarks[sellerAssetPair] = balances[sellerAssetPair]
		}

		var buyerAccountID string
		if buyerAccountID, ok = accountMap[rawBuyer]; !ok {
			buyerAccountID = assettest.CreateAccountFixture(ctx, t, mnodeID, rawBuyer, nil)
			accountMap[rawBuyer] = buyerAccountID
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
	balances = make(map[accountAssetPair]int64)
	for accountAsset, lowWaterMark := range lowWaterMarks {
		if lowWaterMark >= 0 {
			continue
		}
		assettest.IssueAssetsFixture(ctx, t, accountAsset.assetID, uint64(-lowWaterMark), accountAsset.accountID)
		balances[accountAsset] = -lowWaterMark
	}

	b, err := generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	prevBlockTimestamp := b.Timestamp

	type spotCheck struct {
		timestamp uint64
		assetID   bc.AssetID
		accountID string
		amount    uint64
	}
	var spotChecks []*spotCheck

	// Tracks balance deltas in between blocks
	newBalances := make(balanceMap)

	// Execute the trades
	for i, trade := range trades {
		s := []*txbuilder.Source{
			asset.NewAccountSource(ctx, &bc.AssetAmount{AssetID: trade.shareAssetID, Amount: trade.shares}, trade.shareSellerID, nil, nil, nil),
			asset.NewAccountSource(ctx, &bc.AssetAmount{AssetID: usdAssetID, Amount: trade.dollars}, trade.shareBuyerID, nil, nil, nil),
		}
		d := []*txbuilder.Destination{
			assettest.AccountDest(ctx, t, trade.shareSellerID, usdAssetID, trade.dollars),
			assettest.AccountDest(ctx, t, trade.shareBuyerID, trade.shareAssetID, trade.shares),
		}
		assettest.Transfer(ctx, t, s, d)

		newBalances[accountAssetPair{trade.shareSellerID, usdAssetID}] += int64(trade.dollars)
		newBalances[accountAssetPair{trade.shareSellerID, trade.shareAssetID}] -= int64(trade.shares)

		newBalances[accountAssetPair{trade.shareBuyerID, usdAssetID}] -= int64(trade.dollars)
		newBalances[accountAssetPair{trade.shareBuyerID, trade.shareAssetID}] += int64(trade.shares)

		// Land a block every ten trades or so
		if i == len(trades)-1 || rand.Intn(10) == 0 {
			b, err = generator.MakeBlock(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if b.Timestamp > prevBlockTimestamp {
				// Now safe to snapshot balances as of prevBlockTimestamp
				for accountAsset, balance := range balances {
					if rand.Intn(len(balances)) < 4 {
						s := &spotCheck{
							timestamp: prevBlockTimestamp,
							assetID:   accountAsset.assetID,
							accountID: accountAsset.accountID,
							amount:    uint64(balance),
						}
						spotChecks = append(spotChecks, s)
					}
				}
			}
			prevBlockTimestamp = b.Timestamp

			for accountAsset, balance := range newBalances {
				balances[accountAsset] += balance
			}
			newBalances = make(balanceMap)
		}
	}

	for i, s := range spotChecks {
		ts := time.Unix(int64(s.timestamp), 0)
		sums, _, err := HistoricalBalancesByAccount(ctx, s.accountID, ts, &s.assetID, "", 0)
		if err != nil {
			t.Fatal(err)
		}
		var sum uint64
		if len(sums) == 1 {
			sum = sums[0].Amount
		} else if len(sums) != 0 {
			t.Fatal(fmt.Errorf("expected 0 results or 1 from HistoricalBalancesByAccount(%s, %s), got %d", s.accountID, s.assetID, len(sums)))
		}
		if sum != s.amount {
			t.Errorf("Spot check %d of %d: Got %d units of %s in account %s at time %s, expected %d", i+1, len(spotChecks), sum, s.assetID, s.accountID, ts, s.amount)
		}
	}
}

func mustParseInt(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return n
}
