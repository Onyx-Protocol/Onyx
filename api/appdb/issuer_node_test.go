package appdb_test

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	. "chain/api/appdb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
)

func TestInsertIssuerNode(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		_ = newTestIssuerNode(t, ctx, nil, "foo")
	})
}

func TestListIssuerNodes(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0'),
			('proj-id-1', 'proj-1');

		INSERT INTO issuer_nodes
			(id, project_id, key_index, keyset, label, created_at, archived)
		VALUES
			-- insert in reverse chronological order, to ensure that ListIssuerNodes
			-- is performing a sort.
			('in-id-2', 'proj-id-1', 2, '{}', 'in-2', now(), false),
			('in-id-1', 'proj-id-0', 1, '{}', 'in-1', now(), false),
			('in-id-0', 'proj-id-0', 0, '{}', 'in-0', now(), false),
			('in-id-3', 'proj-id-0', 3, '{}', 'in-3-archived', now(), true);
	`
	withContext(t, sql, func(ctx context.Context) {
		examples := []struct {
			projID string
			want   []*IssuerNode
		}{
			{
				"proj-id-0",
				[]*IssuerNode{
					{ID: "in-id-0", Label: "in-0"},
					{ID: "in-id-1", Label: "in-1"},
				},
			},
			{
				"proj-id-1",
				[]*IssuerNode{
					{ID: "in-id-2", Label: "in-2"},
				},
			},
		}

		for _, ex := range examples {
			t.Log("Example:", ex.projID)

			got, err := ListIssuerNodes(ctx, ex.projID)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("issuer nodes:\ngot:  %v\nwant: %v", got, ex.want)
			}
		}
	})
}

func TestGetIssuerNodes(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		proj := newTestProject(t, ctx, "foo", nil)
		in, err := InsertIssuerNode(ctx, proj.ID, "in-0", []*hdkey.XKey{dummyXPub}, []*hdkey.XKey{dummyXPrv}, 1)
		if err != nil {
			t.Fatalf("unexpected error on InsertIssuerNode: %v", err)
		}
		examples := []struct {
			id      string
			want    *IssuerNode
			wantErr error
		}{
			{
				in.ID,
				&IssuerNode{
					ID:    in.ID,
					Label: "in-0",
					Keys: []*NodeKey{
						{
							Type: "node",
							XPub: dummyXPub,
							XPrv: dummyXPrv,
						},
					},
					SigsReqd: 1,
				},
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

			got, gotErr := GetIssuerNode(ctx, ex.id)

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("issuer node:\ngot:  %v\nwant: %v", got, ex.want)
			}

			if errors.Root(gotErr) != ex.wantErr {
				t.Errorf("get issuer node error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
			}
		}
	})
}

func TestUpdateIssuerNode(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		issuerNode := newTestIssuerNode(t, ctx, nil, "foo")

		newLabel := "bar"

		err := UpdateIssuerNode(ctx, issuerNode.ID, &newLabel)
		if err != nil {
			t.Errorf("update issuer node error %v", err)
		}

		issuerNode, err = GetIssuerNode(ctx, issuerNode.ID)
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
	withContext(t, "", func(ctx context.Context) {
		issuerNode := newTestIssuerNode(t, ctx, nil, "foo")
		err := UpdateIssuerNode(ctx, issuerNode.ID, nil)
		if err != nil {
			t.Errorf("update issuer node error %v", err)
		}

		issuerNode, err = GetIssuerNode(ctx, issuerNode.ID)
		if err != nil {
			t.Fatalf("could not get issuer node with id %s: %v:", issuerNode.ID, err)
		}
		if issuerNode.Label != "foo" {
			t.Errorf("expected foo, got %s", issuerNode.Label)
		}
	})
}

func TestArchiveIssuerNode(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		issuerNode := newTestIssuerNode(t, ctx, nil, "foo")
		asset := newTestAsset(t, ctx, issuerNode)
		err := ArchiveIssuerNode(ctx, issuerNode.ID)
		if err != nil {
			t.Errorf("could not archive issuer node with id %s: %v", issuerNode.ID, err)
		}

		var archived bool
		checkQ := `SELECT archived FROM issuer_nodes WHERE id = $1`
		err = pg.FromContext(ctx).QueryRow(ctx, checkQ, issuerNode.ID).Scan(&archived)

		if !archived {
			t.Errorf("expected issuer node %s to be archived", issuerNode.ID)
		}

		checkAssetQ := `SELECT archived FROM assets WHERE id = $1`
		err = pg.FromContext(ctx).QueryRow(ctx, checkAssetQ, asset.Hash.String()).Scan(&archived)
		if !archived {
			t.Errorf("expected child asset %s to be archived", asset.Hash.String())
		}
	})
}
