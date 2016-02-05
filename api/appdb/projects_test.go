package appdb_test

import (
	"database/sql"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	. "chain/api/appdb"
	"chain/api/asset/assettest"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestCreateProject(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	userID := assettest.CreateUserFixture(ctx, t, "user@chain.com", "password")
	p, err := CreateProject(ctx, "new-proj", userID)
	if err != nil {
		t.Fatal(err)
	}

	if p.ID == "" {
		t.Error("project ID is blank")
	}
	if p.Name != "new-proj" {
		t.Errorf("project name = %v want new-proj", p.Name)
	}

	// Make sure the user was set as an admin.
	role, err := checkRole(ctx, p.ID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if role != "admin" {
		t.Errorf("user role = %v want admin", role)
	}
}

func TestListProjects(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	userID1 := assettest.CreateUserFixture(ctx, t, "foo@chain.com", "password")
	userID2 := assettest.CreateUserFixture(ctx, t, "bar@chain.com", "password")
	userID3 := assettest.CreateUserFixture(ctx, t, "baz@chain.com", "password")
	projectID1 := assettest.CreateProjectFixture(ctx, t, userID1, "first project")
	projectID2 := assettest.CreateProjectFixture(ctx, t, userID1, "second project")
	projectID3 := assettest.CreateProjectFixture(ctx, t, userID3, "third project")
	assettest.CreateMemberFixture(ctx, t, userID1, projectID3, "developer")
	assettest.CreateMemberFixture(ctx, t, userID2, projectID1, "developer")
	if err := ArchiveProject(ctx, projectID3); err != nil {
		t.Fatal(err)
	}

	examples := []struct {
		userID string
		want   []*Project
	}{
		{
			userID1,
			[]*Project{
				{projectID1, "first project"},
				{projectID2, "second project"},
			},
		},
		{
			userID2,
			[]*Project{
				{projectID1, "first project"},
			},
		},
		{
			userID3,
			nil,
		},
	}

	for _, ex := range examples {
		t.Log("user:", ex.userID)

		got, err := ListProjects(ctx, ex.userID)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("projects:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestGetProject(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	projectID1 := assettest.CreateProjectFixture(ctx, t, "", "first project")
	projectID2 := assettest.CreateProjectFixture(ctx, t, "", "second project")
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
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	projectID1 := assettest.CreateProjectFixture(ctx, t, "", "first project")

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
			_ = pg.FromContext(ctx).QueryRow(ctx, q, ex.id).Scan(&got)
			if got != "new-name" {
				t.Errorf("name got=%v want new-name", got)
			}
		}
	}
}

func TestArchiveProject(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	userID1 := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "password")
	userID2 := assettest.CreateUserFixture(ctx, t, "baz@bar.com", "password")
	projectID1 := assettest.CreateProjectFixture(ctx, t, userID1, "first project")
	projectID2 := assettest.CreateProjectFixture(ctx, t, "", "second project")
	projectID3 := assettest.CreateProjectFixture(ctx, t, "", "third project")
	assettest.CreateMemberFixture(ctx, t, userID1, projectID2, "developer")
	assettest.CreateMemberFixture(ctx, t, userID1, projectID3, "developer")
	assettest.CreateMemberFixture(ctx, t, userID2, projectID1, "developer")
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

		err := ArchiveProject(ctx, ex.id)
		if errors.Root(err) != ex.wantErr {
			t.Errorf("error got=%v want=%v", errors.Root(err), ex.wantErr)
		}

		if ex.wantErr == nil {
			// Verify that the project is marked as archived.
			q := `SELECT archived FROM projects WHERE id = $1`
			var got bool
			_ = pg.FromContext(ctx).QueryRow(ctx, q, ex.id).Scan(&got)
			if !got {
				t.Errorf("archived=%v want true", got)
			}

			var count int

			// Check that all manager nodes are archived.
			q = `SELECT COUNT(id) FROM manager_nodes WHERE project_id = $1 AND NOT archived`
			if err := pg.FromContext(ctx).QueryRow(ctx, q, ex.id).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Errorf("manager_nodes count=%v, want 0", count)
			}

			// Check that all issuer nodes are archived.
			q = `SELECT COUNT(id) FROM issuer_nodes WHERE project_id = $1 AND NOT archived`
			if err := pg.FromContext(ctx).QueryRow(ctx, q, ex.id).Scan(&count); err != nil {
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
			if err := pg.FromContext(ctx).QueryRow(ctx, q, ex.id).Scan(&count); err != nil {
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
			if err := pg.FromContext(ctx).QueryRow(ctx, q, ex.id).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Errorf("assets count=%v, want 0", count)
			}
		}
	}
}

func TestListMembers(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	userID1 := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "password")
	userID2 := assettest.CreateUserFixture(ctx, t, "baz@bar.com", "password")
	projectID1 := assettest.CreateProjectFixture(ctx, t, userID1, "")
	projectID2 := assettest.CreateProjectFixture(ctx, t, userID2, "")
	assettest.CreateMemberFixture(ctx, t, userID2, projectID1, "developer")

	examples := []struct {
		projectID string
		want      []*Member
	}{
		{
			projectID1,
			[]*Member{
				{userID2, "baz@bar.com", "developer"},
				{userID1, "foo@bar.com", "admin"},
			},
		},
		{
			projectID2,
			[]*Member{
				{userID2, "baz@bar.com", "admin"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("project:", ex.projectID)

		got, err := ListMembers(ctx, ex.projectID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("members:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestAddMember(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	userID1 := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "password")
	userID2 := assettest.CreateUserFixture(ctx, t, "baz@bar.com", "password")
	userID3 := assettest.CreateUserFixture(ctx, t, "mrbenevolent@dictators.com", "password")
	projectID1 := assettest.CreateProjectFixture(ctx, t, userID1, "")

	if err := AddMember(ctx, projectID1, userID2, "developer"); err != nil {
		t.Fatal(err)
	}

	role, err := checkRole(ctx, projectID1, userID2)
	if err != nil {
		t.Fatal(err)
	}
	if role != "developer" {
		t.Errorf("role = %v want developer", role)
	}

	// Repeated attempts result in error.
	err = AddMember(ctx, projectID1, userID2, "developer")
	if errors.Root(err) != ErrAlreadyMember {
		t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), ErrAlreadyMember)
	}

	// Invalid roles result in error
	err = AddMember(ctx, projectID1, userID3, "benevolent-dictator")
	if errors.Root(err) != ErrBadRole {
		t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), ErrBadRole)
	}
}

func TestUpdateMember(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	userID1 := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "password")
	projectID1 := assettest.CreateProjectFixture(ctx, t, userID1, "")

	err := UpdateMember(ctx, projectID1, userID1, "developer")
	if err != nil {
		t.Fatal(err)
	}

	role, err := checkRole(ctx, projectID1, userID1)
	if err != nil {
		t.Fatal(err)
	}
	if role != "developer" {
		t.Errorf("role = %v want developer", role)
	}

	// Updates for non-existing users result in error.
	err = UpdateMember(ctx, projectID1, "not-a-user-id", "developer")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), pg.ErrUserInputNotFound)
	}

	// Invalid roles result in error
	err = UpdateMember(ctx, projectID1, userID1, "benevolent-dictator")
	if errors.Root(err) != ErrBadRole {
		t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), ErrBadRole)
	}
}

func TestRemoveMember(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	userID1 := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "password")
	userID2 := assettest.CreateUserFixture(ctx, t, "baz@bar.com", "password")

	projectID1 := assettest.CreateProjectFixture(ctx, t, userID1, "a new project")
	assettest.CreateMemberFixture(ctx, t, userID2, projectID1, "developer")

	projectID2 := assettest.CreateProjectFixture(ctx, t, "", "another project")
	assettest.CreateMemberFixture(ctx, t, userID1, projectID2, "developer")

	err := RemoveMember(ctx, projectID1, userID1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = checkRole(ctx, projectID1, userID1)
	if err != sql.ErrNoRows {
		t.Errorf("error = %v want %v", err, sql.ErrNoRows)
	}

	// Shouldn't affect other members
	role, err := checkRole(ctx, projectID1, userID2)
	if err != nil {
		t.Fatal(err)
	}
	if role != "developer" {
		t.Errorf("user2 role in proj1 = %v want developer", role)
	}

	// Shouldn't affect other projects
	role, err = checkRole(ctx, projectID2, userID1)
	if err != nil {
		t.Fatal(err)
	}
	if role != "developer" {
		t.Errorf("user 1 role in project 2 = %v want developer", role)
	}
}

func checkRole(ctx context.Context, projID, userID string) (string, error) {
	var (
		q = `
			SELECT role
			FROM members
			WHERE project_id = $1 AND user_id = $2
		`
		role string
	)
	err := pg.FromContext(ctx).QueryRow(ctx, q, projID, userID).Scan(&role)
	return role, err
}
