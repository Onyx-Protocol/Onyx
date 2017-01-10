package query

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/pin"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/mempool"
	"chain/protocol/prottest"
)

func setupQueryTest(t *testing.T) (context.Context, *Indexer, time.Time, time.Time, string, string, bc.AssetID, bc.AssetID) {
	time1 := time.Now()

	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	indexer := NewIndexer(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	assets := asset.NewRegistry(db, c, pinStore)
	assets.IndexAssets(indexer)
	indexer.RegisterAnnotator(accounts.AnnotateTxs)
	indexer.RegisterAnnotator(assets.AnnotateTxs)
	go assets.ProcessBlocks(ctx)
	go indexer.ProcessBlocks(ctx)

	acct1 := coretest.CreateAccount(ctx, t, accounts, "", nil)
	acct2 := coretest.CreateAccount(ctx, t, accounts, "", nil)

	asset1Tags := map[string]interface{}{"currency": "USD"}

	coretest.CreateAsset(ctx, t, assets, nil, "", asset1Tags)

	asset1 := coretest.CreateAsset(ctx, t, assets, nil, "", asset1Tags)
	asset2 := coretest.CreateAsset(ctx, t, assets, nil, "", nil)

	p := mempool.New()
	coretest.IssueAssets(ctx, t, c, p, assets, accounts, asset1, 867, acct1)
	coretest.IssueAssets(ctx, t, c, p, assets, accounts, asset2, 100, acct1)

	prottest.MakeBlock(t, c, p.Dump(ctx))
	<-pinStore.PinWaiter(TxPinName, c.Height())

	time2 := time.Now()

	return ctx, indexer, time1, time2, acct1, acct2, asset1, asset2
}
