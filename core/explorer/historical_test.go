package explorer

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/core/asset/assettest"
	"chain/core/generator"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func BenchmarkWithHistoricalOutputs(b *testing.B) {
	benchmarkHistoricalOutputs(b, true)
}

func BenchmarkWithoutHistoricalOutputs(b *testing.B) {
	benchmarkHistoricalOutputs(b, false)
}

func benchmarkHistoricalOutputs(b *testing.B, historicalOutputs bool) {
	for n := 0; n < b.N; n++ {
		ctx := context.Background()
		dbtx := pgtest.NewTx(b)
		dbctx := pg.NewContext(ctx, dbtx)
		fc, err := assettest.InitializeSigningGenerator(dbctx, nil, nil)
		if err != nil {
			b.Fatal(err)
		}

		// side effect: register Explorer as a fc block callback
		New(fc, dbtx, nil, nil, 0, historicalOutputs, true)

		n := 0

		populateCallbacks := &assettest.PopulateCallbacks{
			Trade: func(sellerID, buyerID string, shareAssetID, usdAssetID bc.AssetID, shares, dollars uint64) {
				n++
				if n%10 == 0 {
					_, err := generator.MakeBlock(dbctx)
					if err != nil {
						b.Fatal(err)
					}
				}
			},
		}
		assettest.Populate(dbctx, b, "glittercosmall.csv", populateCallbacks)
	}
}

func TestHistoricalOutputs(t *testing.T) {
	ctx := context.Background()
	dbtx := pgtest.NewTx(t)
	dbctx := pg.NewContext(ctx, dbtx)
	fc, err := assettest.InitializeSigningGenerator(dbctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	e := New(fc, dbtx, nil, nil, 0, true, true)

	type (
		spotCheck struct {
			timestampMS uint64
			assetID     bc.AssetID
			accountID   string
			amount      uint64
		}
		accountAssetPair struct {
			accountID string
			assetID   bc.AssetID
		}
		balanceMap map[accountAssetPair]uint64
	)
	var (
		spotChecks           []*spotCheck
		prevBlockTimestampMS uint64
		nTrades              int
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
			b, err := generator.MakeBlock(dbctx)
			if err != nil {
				t.Fatal(err)
			}
			prevBlockTimestampMS = b.TimestampMS
		},
		Trade: func(sellerID, buyerID string, shareAssetID, usdAssetID bc.AssetID, shares, dollars uint64) {
			tradeNum++

			currentBalances[accountAssetPair{sellerID, shareAssetID}] -= shares
			currentBalances[accountAssetPair{sellerID, usdAssetID}] += dollars
			currentBalances[accountAssetPair{buyerID, shareAssetID}] += shares
			currentBalances[accountAssetPair{buyerID, usdAssetID}] -= dollars

			if tradeNum == nTrades || rand.Intn(10) == 0 {
				b, err := generator.MakeBlock(dbctx)
				if err != nil {
					t.Fatal(err)
				}
				if b.TimestampMS > prevBlockTimestampMS {
					// Now safe to snapshot balances as of prevBlockTimestamp
					for accountAsset, balance := range prevBlockBalances {
						if rand.Intn(len(prevBlockBalances)) < 4 {
							s := &spotCheck{
								timestampMS: prevBlockTimestampMS,
								assetID:     accountAsset.assetID,
								accountID:   accountAsset.accountID,
								amount:      balance,
							}
							spotChecks = append(spotChecks, s)
						}
					}
				}
				prevBlockTimestampMS = b.TimestampMS
				for k, v := range currentBalances {
					prevBlockBalances[k] = v
				}
			}
		},
	}
	assettest.Populate(dbctx, t, "glittercosmall.csv", populateCallbacks)

	// Perform spot-checks
	for i, s := range spotChecks {
		ts := time.Unix(0, int64(s.timestampMS)*int64(time.Millisecond))
		sums, _, err := e.HistoricalBalancesByAccount(ctx, s.accountID, ts, &s.assetID, "", 0)
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

					sums, last, err = e.HistoricalBalancesByAccount(ctx, accountID, time.Now(), nil, last, 1)
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
