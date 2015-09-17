package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
)

func TestCreateAssetGroup(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	id, err := CreateAssetGroup(ctx, "a1", "foo", []*hdkey.XKey{dummyXPub})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if id == "" {
		t.Errorf("got empty asset group id")
	}
}

func TestListAssetGroups(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO applications (id, name) VALUES
			('app-id-0', 'app-0'),
			('app-id-1', 'app-1');

		INSERT INTO asset_groups
			(id, application_id, key_index, keyset, label, created_at)
		VALUES
			-- insert in reverse chronological order, to ensure that ListAssetGroups
			-- is performing a sort.
			('ag-id-0', 'app-id-0', 0, '{}', 'ag-0', now()),
			('ag-id-1', 'app-id-0', 1, '{}', 'ag-1', now() - '1 day'::interval),

			('ag-id-2', 'app-id-1', 2, '{}', 'ag-2', now());
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		appID string
		want  []*AssetGroup
	}{
		{
			"app-id-0",
			[]*AssetGroup{
				{ID: "ag-id-1", Blockchain: "sandbox", Label: "ag-1"},
				{ID: "ag-id-0", Blockchain: "sandbox", Label: "ag-0"},
			},
		},
		{
			"app-id-1",
			[]*AssetGroup{
				{ID: "ag-id-2", Blockchain: "sandbox", Label: "ag-2"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.appID)

		got, err := ListAssetGroups(ctx, ex.appID)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("asset groups:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestGetAssetGroup(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO applications (id, name) VALUES
			('app-id-0', 'app-0');

		INSERT INTO asset_groups (id, application_id, key_index, keyset, label) VALUES
			('ag-id-0', 'app-id-0', 0, '{}', 'ag-0');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		id      string
		want    *AssetGroup
		wantErr error
	}{
		{
			"ag-id-0",
			&AssetGroup{ID: "ag-id-0", Label: "ag-0", Blockchain: "sandbox"},
			nil,
		},
		{
			"nonexistent",
			nil,
			pg.ErrUserInputNotFound,
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.id)

		got, gotErr := GetAssetGroup(ctx, ex.id)

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("asset group:\ngot:  %v\nwant: %v", got, ex.want)
		}

		if errors.Root(gotErr) != ex.wantErr {
			t.Errorf("get asset group error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
		}
	}
}
