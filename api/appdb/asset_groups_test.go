package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
)

func TestInsertAssetGroup(t *testing.T) {
	withContext(t, "", func(t *testing.T, ctx context.Context) {
		_ = newTestIssuerNode(t, ctx, "a1", "foo")
	})
}

func TestListAssetGroups(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0'),
			('proj-id-1', 'proj-1');

		INSERT INTO issuer_nodes
			(id, project_id, key_index, keyset, label, created_at)
		VALUES
			-- insert in reverse chronological order, to ensure that ListAssetGroups
			-- is performing a sort.
			('ag-id-0', 'proj-id-0', 0, '{}', 'ag-0', now()),
			('ag-id-1', 'proj-id-0', 1, '{}', 'ag-1', now() - '1 day'::interval),

			('ag-id-2', 'proj-id-1', 2, '{}', 'ag-2', now());
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		examples := []struct {
			projID string
			want   []*AssetGroup
		}{
			{
				"proj-id-0",
				[]*AssetGroup{
					{ID: "ag-id-1", Blockchain: "sandbox", Label: "ag-1"},
					{ID: "ag-id-0", Blockchain: "sandbox", Label: "ag-0"},
				},
			},
			{
				"proj-id-1",
				[]*AssetGroup{
					{ID: "ag-id-2", Blockchain: "sandbox", Label: "ag-2"},
				},
			},
		}

		for _, ex := range examples {
			t.Log("Example:", ex.projID)

			got, err := ListAssetGroups(ctx, ex.projID)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("asset groups:\ngot:  %v\nwant: %v", got, ex.want)
			}
		}
	})
}

func TestGetAssetGroup(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0');

		INSERT INTO issuer_nodes (id, project_id, key_index, keyset, label) VALUES
			('ag-id-0', 'proj-id-0', 0, '{}', 'ag-0');
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
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
	})
}

func TestUpdateIssuerNode(t *testing.T) {
	withContext(t, "", func(t *testing.T, ctx context.Context) {
		issuerNode := newTestIssuerNode(t, ctx, "a1", "foo")

		newLabel := "bar"

		err := UpdateIssuerNode(ctx, issuerNode.ID, &newLabel)
		if err != nil {
			t.Errorf("update issuer node error %v", err)
		}

		issuerNode, err = GetAssetGroup(ctx, issuerNode.ID)
		if err != nil {
			t.Fatalf("could not get issuer node with id %s: %v", issuerNode.ID, err)
		}
		if issuerNode.Label != newLabel {
			t.Errorf("expected %s, got %s", newLabel, issuerNode.Label)
		}
	})
}

// Test that calling UpdateIssuerNode with no new label is a no-op.
func TestUpdateIssuerNodeNoUpdate(t *testing.T) {
	withContext(t, "", func(t *testing.T, ctx context.Context) {
		issuerNode := newTestIssuerNode(t, ctx, "a1", "foo")
		err := UpdateIssuerNode(ctx, issuerNode.ID, nil)
		if err != nil {
			t.Errorf("update issuer node error %v", err)
		}

		issuerNode, err = GetAssetGroup(ctx, issuerNode.ID)
		if err != nil {
			t.Fatalf("could not get issuer node with id %s: %v:", issuerNode.ID, err)
		}
		if issuerNode.Label != "foo" {
			t.Errorf("expected foo, got %s", issuerNode.Label)
		}
	})
}
