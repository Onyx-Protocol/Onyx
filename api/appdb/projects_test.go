package appdb

import (
	"database/sql"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
)

// A common fixture for use in other test files within the package.
const (
	sampleProjectFixture string = `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
	`

	// A fixture for tests within this file.
	projectsFixtures = `
		INSERT INTO users (id, email, password_hash) VALUES
			('user-id-0', 'foo@bar.com', 'password-does-not-matter'),
			('user-id-1', 'baz@bar.com', 'password-does-not-matter'),
			('user-id-2', 'biz@bar.com', 'password-does-not-matter');

		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0'),
			('proj-id-1', 'proj-1');

		INSERT INTO members (project_id, user_id, role) VALUES
			('proj-id-0', 'user-id-0', 'admin'),
			('proj-id-1', 'user-id-0', 'developer'),
			('proj-id-0', 'user-id-1', 'developer');
	`
)

func TestCreateProject(t *testing.T) {
	withContext(t, projectsFixtures, func(t *testing.T, ctx context.Context) {
		p, err := CreateProject(ctx, "new-proj", "user-id-0")
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
		role, err := checkRole(ctx, p.ID, "user-id-0")
		if err != nil {
			t.Fatal(err)
		}

		if role != "admin" {
			t.Errorf("user role = %v want admin", role)
		}
	})
}

func TestListProjects(t *testing.T) {
	withContext(t, projectsFixtures, func(t *testing.T, ctx context.Context) {
		examples := []struct {
			userID string
			want   []*Project
		}{
			{
				"user-id-0",
				[]*Project{
					{"proj-id-0", "proj-0"},
					{"proj-id-1", "proj-1"},
				},
			},
			{
				"user-id-1",
				[]*Project{
					{"proj-id-0", "proj-0"},
				},
			},
			{
				"user-id-2",
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
	})
}

func TestGetProject(t *testing.T) {
	withContext(t, projectsFixtures, func(t *testing.T, ctx context.Context) {
		examples := []struct {
			id          string
			wantProject *Project
			wantErr     error
		}{
			{"proj-id-0", &Project{ID: "proj-id-0", Name: "proj-0"}, nil},
			{"proj-id-1", &Project{ID: "proj-id-1", Name: "proj-1"}, nil},
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
	})
}

func TestUpdateProject(t *testing.T) {
	withContext(t, projectsFixtures, func(t *testing.T, ctx context.Context) {
		examples := []struct {
			id      string
			wantErr error
		}{
			{"proj-id-0", nil},
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
				_ = pg.FromContext(ctx).QueryRow(q, ex.id).Scan(&got)
				if got != "new-name" {
					t.Errorf("name got=%v want new-name", got)
				}
			}
		}
	})
}

func TestListMembers(t *testing.T) {
	withContext(t, projectsFixtures, func(t *testing.T, ctx context.Context) {
		examples := []struct {
			projectID string
			want      []*Member
		}{
			{
				"proj-id-0",
				[]*Member{
					{"user-id-1", "baz@bar.com", "developer"},
					{"user-id-0", "foo@bar.com", "admin"},
				},
			},
			{
				"proj-id-1",
				[]*Member{
					{"user-id-0", "foo@bar.com", "developer"},
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
	})
}

func TestAddMember(t *testing.T) {
	withContext(t, projectsFixtures, func(t *testing.T, ctx context.Context) {
		err := AddMember(ctx, "proj-id-0", "user-id-2", "developer")
		if err != nil {
			t.Fatal(err)
		}

		role, err := checkRole(ctx, "proj-id-0", "user-id-2")
		if err != nil {
			t.Fatal(err)
		}

		if role != "developer" {
			t.Errorf("role = %v want developer", role)
		}

		// Repeated attempts result in error.
		err = AddMember(ctx, "proj-id-0", "user-id-2", "developer")
		if errors.Root(err) != ErrAlreadyMember {
			t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), ErrAlreadyMember)
		}

		// Invalid roles result in error
		err = AddMember(ctx, "proj-id-0", "user-id-3", "benevolent-dictator")
		if errors.Root(err) != ErrBadRole {
			t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), ErrBadRole)
		}
	})
}

func TestUpdateMember(t *testing.T) {
	withContext(t, projectsFixtures, func(t *testing.T, ctx context.Context) {
		err := UpdateMember(ctx, "proj-id-0", "user-id-0", "developer")
		if err != nil {
			t.Fatal(err)
		}

		role, err := checkRole(ctx, "proj-id-0", "user-id-0")
		if err != nil {
			t.Fatal(err)
		}

		if role != "developer" {
			t.Errorf("role = %v want developer", role)
		}

		// Updates for non-existing users result in error.
		err = UpdateMember(ctx, "proj-id-0", "user-id-2", "developer")
		if errors.Root(err) != pg.ErrUserInputNotFound {
			t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), pg.ErrUserInputNotFound)
		}

		// Invalid roles result in error
		err = UpdateMember(ctx, "proj-id-0", "user-id-0", "benevolent-dictator")
		if errors.Root(err) != ErrBadRole {
			t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), ErrBadRole)
		}
	})
}

func TestRemoveMember(t *testing.T) {
	withContext(t, projectsFixtures, func(t *testing.T, ctx context.Context) {
		err := RemoveMember(ctx, "proj-id-0", "user-id-0")
		if err != nil {
			t.Fatal(err)
		}

		_, err = checkRole(ctx, "proj-id-0", "user-id-0")
		if err != sql.ErrNoRows {
			t.Errorf("error = %v want %v", err, sql.ErrNoRows)
		}

		// Shouldn't affect other members
		role, err := checkRole(ctx, "proj-id-0", "user-id-1")
		if err != nil {
			t.Fatal(err)
		}

		if role != "developer" {
			t.Errorf("user-1 role in proj-0 = %v want developer", role)
		}

		// Shouldn't affect other projects
		role, err = checkRole(ctx, "proj-id-1", "user-id-0")
		if err != nil {
			t.Fatal(err)
		}

		if role != "developer" {
			t.Errorf("user-0 role in proj-1 = %v want developer", role)
		}
	})
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
	err := pg.FromContext(ctx).QueryRow(q, projID, userID).Scan(&role)
	return role, err
}
