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
