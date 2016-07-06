package appdb_test

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	. "chain/core/appdb"
	"chain/core/asset/assettest"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/testutil"
)

func TestCreateProject(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	p, err := CreateProject(ctx, "new-proj")
	if err != nil {
		t.Fatal(err)
	}

	if p.ID == "" {
		t.Error("project ID is blank")
	}
	if p.Name != "new-proj" {
		t.Errorf("project name = %v want new-proj", p.Name)
	}
}

func TestListProjects(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	projectID1 := assettest.CreateProjectFixture(ctx, t, "first project")
	projectID2 := assettest.CreateProjectFixture(ctx, t, "second project")
	projectID3 := assettest.CreateProjectFixture(ctx, t, "third project")
	if err := ArchiveProject(ctx, projectID3); err != nil {
		t.Fatal(err)
	}

	examples := []struct {
		want []*Project
	}{
		{
			[]*Project{
				{projectID1, "first project"},
				{projectID2, "second project"},
			},
		},
	}

	for _, ex := range examples {
		got, err := ListProjects(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("projects:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestGetProject(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	projectID1 := assettest.CreateProjectFixture(ctx, t, "first project")
	projectID2 := assettest.CreateProjectFixture(ctx, t, "second project")
	examples := []struct {
		id          string
		wantProject *Project
		wantErr     error
	}{
		{projectID1, &Project{ID: projectID1, Name: "first project"}, nil},
		{projectID2, &Project{ID: projectID2, Name: "second project"}, nil},
		{"nonexistent", nil, pg.ErrUserInputNotFound},
	}

	for _, ex := range examples {
		t.Log("project:", ex.id)
		gotProject, gotErr := GetProject(ctx, ex.id)

		if !reflect.DeepEqual(gotProject, ex.wantProject) {
			t.Errorf("project:\ngot:  %v\nwant: %v", gotProject, ex.wantProject)
		}
		if errors.Root(gotErr) != ex.wantErr {
			t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
		}
	}
}

func TestUpdateProject(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	projectID1 := assettest.CreateProjectFixture(ctx, t, "first project")

	examples := []struct {
		id      string
		wantErr error
	}{
		{projectID1, nil},
		{"nonexistent", pg.ErrUserInputNotFound},
	}

	for _, ex := range examples {
		t.Log("project:", ex.id)

		err := UpdateProject(ctx, ex.id, "new-name")
		if errors.Root(err) != ex.wantErr {
			t.Errorf("error got=%v want=%v", errors.Root(err), ex.wantErr)
		}
		if ex.wantErr == nil {
			q := `SELECT name FROM projects WHERE id = $1`
			var got string
			_ = pg.QueryRow(ctx, q, ex.id).Scan(&got)
			if got != "new-name" {
				t.Errorf("name got=%v want new-name", got)
			}
		}
	}
}

func TestArchiveProject(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	projectID1 := assettest.CreateProjectFixture(ctx, t, "first project")
	projectID2 := assettest.CreateProjectFixture(ctx, t, "second project")
	inodeID1 := assettest.CreateIssuerNodeFixture(ctx, t, projectID1, "", nil, nil)
	inodeID2 := assettest.CreateIssuerNodeFixture(ctx, t, projectID1, "", nil, nil)
	mnodeID1 := assettest.CreateManagerNodeFixture(ctx, t, projectID1, "", nil, nil)
	mnodeID2 := assettest.CreateManagerNodeFixture(ctx, t, projectID2, "", nil, nil)
	assettest.CreateAccountFixture(ctx, t, mnodeID1, "", nil)
	assettest.CreateAccountFixture(ctx, t, mnodeID1, "", nil)
	assettest.CreateAccountFixture(ctx, t, mnodeID2, "", nil)
	assettest.CreateAssetFixture(ctx, t, inodeID1, "", "")
	assettest.CreateAssetFixture(ctx, t, inodeID1, "", "")
	assettest.CreateAssetFixture(ctx, t, inodeID2, "", "")

	examples := []struct {
		id      string
		wantErr error
	}{
		{projectID1, nil},
		{"nonexistent", pg.ErrUserInputNotFound},
	}

	for _, ex := range examples {
		t.Log("project:", ex.id)
		_, ctx, err := pg.Begin(ctx)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		err = ArchiveProject(ctx, ex.id)
		if errors.Root(err) != ex.wantErr {
			t.Errorf("error got=%v want=%v", errors.Root(err), ex.wantErr)
		}

		if ex.wantErr == nil {
			// Verify that the project is marked as archived.
			q := `SELECT archived FROM projects WHERE id = $1`
			var got bool
			_ = pg.QueryRow(ctx, q, ex.id).Scan(&got)
			if !got {
				t.Errorf("archived=%v want true", got)
			}

			var count int

			// Check that all manager nodes are archived.
			q = `SELECT COUNT(id) FROM manager_nodes WHERE project_id = $1 AND NOT archived`
			if err := pg.QueryRow(ctx, q, ex.id).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Errorf("manager_nodes count=%v, want 0", count)
			}

			// Check that all issuer nodes are archived.
			q = `SELECT COUNT(id) FROM issuer_nodes WHERE project_id = $1 AND NOT archived`
			if err := pg.QueryRow(ctx, q, ex.id).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Errorf("issuer_nodes count=%v, want 0", count)
			}

			// Check that all accounts are archived.
			q = `
				SELECT COUNT(id) FROM accounts WHERE manager_node_id IN (
					SELECT id FROM manager_nodes WHERE project_id = $1
				) AND NOT archived
			`
			if err := pg.QueryRow(ctx, q, ex.id).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Errorf("accounts count=%v, want 0", count)
			}

			// Check that all assets are archived.
			q = `
				SELECT COUNT(id) FROM assets WHERE issuer_node_id IN (
					SELECT id FROM issuer_nodes WHERE project_id = $1
				) AND NOT archived
			`
			if err := pg.QueryRow(ctx, q, ex.id).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Errorf("assets count=%v, want 0", count)
			}
		}
	}
}
