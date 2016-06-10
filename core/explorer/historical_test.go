package explorer

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"chain/core/asset/assettest"
	"chain/core/generator"
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

	type (
		spotCheck struct {
			timestamp uint64
			assetID   bc.AssetID
			accountID string
			amount    uint64
		}
		accountAssetPair struct {
			accountID string
			assetID   bc.AssetID
		}
		balanceMap map[accountAssetPair]uint64
	)
	var (
		spotChecks         []*spotCheck
		prevBlockTimestamp uint64
		nTrades            int
	)

	// Only updates after landing a block
	prevBlockBalances := make(balanceMap)

	// Updates after every trade
	currentBalances := make(balanceMap)

	var tradeNum int

	populateCallbacks := &assettest.PopulateCallbacks{
		Issue: func(assetID bc.AssetID, accountID string, amount uint64) {
			accountAsset := accountAssetPair{accountID, assetID}
			prevBlockBalances[accountAsset] = amount
			currentBalances[accountAsset] = amount
		},
		AfterIssue: func(n int) {
			nTrades = n
			b, err := generator.MakeBlock(ctx)
			if err != nil {
				t.Fatal(err)
			}
			prevBlockTimestamp = b.Timestamp
		},
		Trade: func(sellerID, buyerID string, shareAssetID, usdAssetID bc.AssetID, shares, dollars uint64) {
			tradeNum++

			currentBalances[accountAssetPair{sellerID, shareAssetID}] -= shares
			currentBalances[accountAssetPair{sellerID, usdAssetID}] += dollars
			currentBalances[accountAssetPair{buyerID, shareAssetID}] += shares
			currentBalances[accountAssetPair{buyerID, usdAssetID}] -= dollars

			if tradeNum == nTrades || rand.Intn(10) == 0 {
				b, err := generator.MakeBlock(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if b.Timestamp > prevBlockTimestamp {
					// Now safe to snapshot balances as of prevBlockTimestamp
					for accountAsset, balance := range prevBlockBalances {
						if rand.Intn(len(prevBlockBalances)) < 4 {
							s := &spotCheck{
								timestamp: prevBlockTimestamp,
								assetID:   accountAsset.assetID,
								accountID: accountAsset.accountID,
								amount:    balance,
							}
							spotChecks = append(spotChecks, s)
						}
					}
				}
				prevBlockTimestamp = b.Timestamp
				for k, v := range currentBalances {
					prevBlockBalances[k] = v
				}
			}
		},
	}
	assettest.Populate(ctx, t, "glittercosmall.csv", populateCallbacks)

	// Perform spot-checks
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

	// Finally, find an account with multiple assets and check that pagination in HistoricalBalancesByAccount works
	assetCountByAccount := make(map[string]int)
	for accountAsset, amount := range currentBalances {
		if amount > 0 {
			accountID := accountAsset.accountID
			assetCountByAccount[accountID]++
			if assetCountByAccount[accountID] > 1 {
				var (
					last  string
					found []bc.AssetAmount
				)
				for {
					var sums []bc.AssetAmount

					sums, last, err = HistoricalBalancesByAccount(ctx, accountID, time.Now(), nil, last, 1)
					if err != nil {
						t.Fatal(err)
					}
					if len(sums) == 0 {
						break
					}
					if len(sums) > 1 {
						t.Fatalf("got %d results from HistoricalBalancesByAccount with a limit of 1", len(sums))
					}
					found = append(found, sums[0])
				}
				// Make sure everything in found is in currentBalances and vice versa
				for _, f := range found {
					accAsset := accountAssetPair{accountID, f.AssetID}
					if f.Amount != currentBalances[accAsset] {
						t.Errorf("found %d units of %s for account %s in database but %d in currentBalances", f.Amount, f.AssetID, accountID, currentBalances[accAsset])
					}
				}
				for accAsset, amt := range currentBalances {
					if amt > 0 && accAsset.accountID == accountID {
						seen := false
						for _, f := range found {
							if f.AssetID == accAsset.assetID {
								seen = true
								if f.Amount != amt {
									t.Errorf("found %d units of %s for account %s in currentBalances but %d in database", amt, f.AssetID, accountID, f.Amount)
								}
								break
							}
						}
						if !seen {
							t.Errorf("found %d units of %s for account %s in currentBalances but nothing in database", amt, accAsset.assetID, accountID)
						}
					}
				}
				break
			}
		}
	}
}
