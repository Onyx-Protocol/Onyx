package core

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"chain/core/account"
	"chain/core/coretest"
	"chain/core/pin"
	"chain/core/query"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestUpdateTags(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	pinStore := pin.NewStore(db)
	indexer := query.NewIndexer(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	accounts.IndexAccounts(indexer)
	api := &API{db: db, chain: c, accounts: accounts, indexer: indexer}

	alias := "test-alias"
	id := coretest.CreateAccount(ctx, t, accounts, alias, map[string]interface{}{
		"test_tag": "v0",
	})

	// Update by ID

	wantTags := map[string]interface{}{
		"test_tag": "v1",
	}

	api.updateAccountTags(ctx, []struct {
		ID    *string
		Alias *string
		Tags  map[string]interface{} `json:"tags"`
	}{
		{
			ID:   &id,
			Tags: wantTags,
		},
	})

	// Lookup via ID and ensure tag was changed

	page, err := api.listAccounts(ctx, requestQuery{
		Filter:       "id=$1",
		FilterParams: []interface{}{id},
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	items := page.Items.([]*query.AnnotatedAccount)
	if len(items) < 1 {
		t.Fatal("result empty")
	}

	gotTags := make(map[string]interface{})
	err = json.Unmarshal([]byte(*items[0].Tags), &gotTags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(gotTags, wantTags) {
		t.Fatalf("tags:\ngot:  %v\nwant: %v", gotTags, wantTags)
	}

	// Lookup via updated tag

	page, err = api.listAccounts(ctx, requestQuery{
		Filter:       "tags.test_tag=$1",
		FilterParams: []interface{}{"v1"},
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	items = page.Items.([]*query.AnnotatedAccount)
	if len(items) < 1 {
		t.Fatal("result empty")
	}

	if items[0].ID != id {
		t.Fatalf("tags:\ngot:  %v\nwant: %v", items[0].ID, id)
	}

	// Update by alias

	wantTags = map[string]interface{}{
		"test_tag": "v2",
	}

	api.updateAccountTags(ctx, []struct {
		ID    *string
		Alias *string
		Tags  map[string]interface{} `json:"tags"`
	}{
		{
			Alias: &alias,
			Tags:  wantTags,
		},
	})

	// Lookup via updated tag

	page, err = api.listAccounts(ctx, requestQuery{
		Filter:       "tags.test_tag=$1",
		FilterParams: []interface{}{"v2"},
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	items = page.Items.([]*query.AnnotatedAccount)
	if len(items) < 1 {
		t.Fatal("result empty")
	}

	if items[0].ID != id {
		t.Fatalf("tags:\ngot:  %v\nwant: %v", items[0].ID, id)
	}
}
