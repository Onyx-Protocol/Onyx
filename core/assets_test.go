package core

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/pin"
	"chain/core/query"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestUpdateAssetTags(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	pinStore := pin.NewStore(db)
	indexer := query.NewIndexer(db, c, pinStore)
	assets := asset.NewRegistry(db, c, pinStore)
	assets.IndexAssets(indexer)
	api := &API{db: db, chain: c, assets: assets, indexer: indexer}

	alias := "test-alias"
	aid := coretest.CreateAsset(ctx, t, assets, nil, alias, map[string]interface{}{"test_tag": "v0"})
	id := aid.String()

	// Update by ID

	wantTags := map[string]interface{}{"test_tag": "v1"}

	api.updateAssetTags(ctx, []struct {
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

	page, err := api.listAssets(ctx, requestQuery{
		Filter:       "id=$1",
		FilterParams: []interface{}{id},
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	items := page.Items.([]*query.AnnotatedAsset)
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

	page, err = api.listAssets(ctx, requestQuery{
		Filter:       "tags.test_tag=$1",
		FilterParams: []interface{}{"v1"},
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	items = page.Items.([]*query.AnnotatedAsset)
	if len(items) < 1 {
		t.Fatal("result empty")
	}

	if items[0].ID.String() != id {
		t.Fatalf("id:\ngot:  %v\nwant: %v", items[0].ID.String(), id)
	}

	// Update by alias

	wantTags = map[string]interface{}{"test_tag": "v2"}

	api.updateAssetTags(ctx, []struct {
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

	page, err = api.listAssets(ctx, requestQuery{
		Filter:       "tags.test_tag=$1",
		FilterParams: []interface{}{"v2"},
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	items = page.Items.([]*query.AnnotatedAsset)
	if len(items) < 1 {
		t.Fatal("result empty")
	}

	if items[0].ID.String() != id {
		t.Fatalf("id:\ngot:  %v\nwant: %v", items[0].ID.String(), id)
	}
}
