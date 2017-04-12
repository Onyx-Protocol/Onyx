package core

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/pin"
	"chain/core/query"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
)

func TestQueryWithClockSkew(t *testing.T) {
	ctx := context.Background()
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	c := prottest.NewChain(t)

	pinStore := pin.NewStore(db)
	err := pinStore.CreatePin(ctx, asset.PinName, 100)
	if err != nil {
		t.Fatal(err)
	}
	err = pinStore.CreatePin(ctx, account.PinName, 100)
	if err != nil {
		t.Fatal(err)
	}
	err = pinStore.CreatePin(ctx, query.TxPinName, 99)
	if err != nil {
		t.Fatal(err)
	}

	indexer := query.NewIndexer(db, c, pinStore)
	api := &API{db: db, chain: c, indexer: indexer}

	tx := legacy.NewTx(legacy.TxData{})
	block := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Height:      100,
			TimestampMS: bc.Millis(time.Now().Add(time.Hour)),
		},
		Transactions: []*legacy.Tx{tx},
	}
	err = indexer.IndexTransactions(ctx, block)
	if err != nil {
		t.Fatal(err)
	}

	p, err := api.listTransactions(ctx, requestQuery{})
	if err != nil {
		t.Fatal(err)
	}
	count := len(p.Items.([]*query.AnnotatedTx))
	if count != 1 {
		t.Errorf("got=%d txs, want %d", count, 1)
	}
}
