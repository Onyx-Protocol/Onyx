package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
)

func TestInsertAdminNode(t *testing.T) {
	withContext(t, "", func(t *testing.T, ctx context.Context) {
		p := newTestProject(t, ctx, "proj", nil)
		_ = newTestAdminNode(t, ctx, p, "foo")

		// Test uniqueness constraint
		_, err := InsertAdminNode(ctx, p.ID, "foo-1")
		if errors.Root(err) != ErrAdminNodeAlreadyExists {
			t.Errorf("err got = %v want %v", errors.Root(err), ErrAdminNodeAlreadyExists)
		}
	})
}

func TestGetAdminNode(t *testing.T) {
	withContext(t, "", func(t *testing.T, ctx context.Context) {
		proj := newTestProject(t, ctx, "foo", nil)
		an, err := InsertAdminNode(ctx, proj.ID, "admin-node-0")
		if err != nil {
			t.Fatalf("unexpected error on InsertAdminNode: %v", err)
		}
		examples := []struct {
			id      string
			want    *AdminNode
			wantErr error
		}{
			{
				an.ID,
				&AdminNode{
					ID:    an.ID,
					Label: "admin-node-0",
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

			got, gotErr := GetAdminNode(ctx, ex.id)

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("adminNode:\ngot:  %v\nwant: %v", got, ex.want)
			}

			if errors.Root(gotErr) != ex.wantErr {
				t.Errorf("get adminNode error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
			}
		}
	})
}

func TestListAdminNodes(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0'),
			('proj-id-1', 'proj-1');

		INSERT INTO admin_nodes (id, project_id, label, created_at) VALUES
			-- insert in reverse chronological order, to ensure that ListAdminNodes
			-- is performing a sort.
			('admin-node-id-0', 'proj-id-0', 'admin-node-0', now()),
			('admin-node-id-1', 'proj-id-0', 'admin-node-1', now() - '1 day'::interval),
			('admin-node-id-2', 'proj-id-1', 'admin-node-2', now());
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		examples := []struct {
			projID string
			want   []*AdminNode
		}{
			{
				"proj-id-0",
				[]*AdminNode{
					{ID: "admin-node-id-1", Label: "admin-node-1"},
					{ID: "admin-node-id-0", Label: "admin-node-0"},
				},
			},
			{
				"proj-id-1",
				[]*AdminNode{
					{ID: "admin-node-id-2", Label: "admin-node-2"},
				},
			},
		}

		for _, ex := range examples {
			t.Log("Example:", ex.projID)

			got, err := ListAdminNodes(ctx, ex.projID)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("adminNodes:\ngot:  %v\nwant: %v", got, ex.want)
			}
		}
	})
}

func TestUpdateAdminNode(t *testing.T) {
	withContext(t, "", func(t *testing.T, ctx context.Context) {
		adminNode := newTestAdminNode(t, ctx, nil, "foo")
		newLabel := "bar"

		err := UpdateAdminNode(ctx, adminNode.ID, &newLabel)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}

		adminNode, err = GetAdminNode(ctx, adminNode.ID)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		if adminNode.Label != newLabel {
			t.Errorf("Expected %s, got %s", newLabel, adminNode.Label)
		}
	})
}

// Test that calling UpdateAdminNode with no new label is a no-op.
func TestUpdateAdminNodeNoUpdate(t *testing.T) {
	withContext(t, "", func(t *testing.T, ctx context.Context) {
		adminNode := newTestAdminNode(t, ctx, nil, "foo")
		err := UpdateAdminNode(ctx, adminNode.ID, nil)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		adminNode, err = GetAdminNode(ctx, adminNode.ID)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		if adminNode.Label != "foo" {
			t.Errorf("Expected foo, got %s", adminNode.Label)
		}
	})
}

func TestDeleteAdminNode(t *testing.T) {
	withContext(t, "", func(t *testing.T, ctx context.Context) {
		adminNode := newTestAdminNode(t, ctx, nil, "foo")

		_, err := GetAdminNode(ctx, adminNode.ID)
		if err != nil {
			t.Errorf("could not get test admin node with id %s: %v", adminNode.ID, err)
		}

		err = DeleteAdminNode(ctx, adminNode.ID)
		if err != nil {
			t.Errorf("could not delete admin node with id %s: %v", adminNode.ID, err)
		}

		_, err = GetAdminNode(ctx, adminNode.ID)
		if errors.Root(err) != pg.ErrUserInputNotFound {
			t.Errorf("unexpected error when trying to get deleted admin node %s: %v", adminNode.ID, err)
		}
	})
}
