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

		INSERT INTO projects (id, name, archived) VALUES
			('proj-id-0', 'proj-0', false),
			('proj-id-1', 'proj-1', false),
			('proj-id-2', 'proj-2', true);

		INSERT INTO members (project_id, user_id, role) VALUES
			('proj-id-0', 'user-id-0', 'admin'),
			('proj-id-1', 'user-id-0', 'developer'),
			('proj-id-2', 'user-id-0', 'developer'),
			('proj-id-0', 'user-id-1', 'developer');
	`
	projectChildrenFixtures = `
		INSERT INTO issuer_nodes (id, project_id, key_index, keyset, label) VALUES
			('in-id-0', 'proj-id-0', 0, '{}', 'in-0'),
			('in-id-1', 'proj-id-0', 1, '{}', 'in-1');

		INSERT INTO assets
			(id, issuer_node_id, key_index, redeem_script, issuance_script, label, sort_id, definition)
		VALUES
			('0000000000000000000000000000000000000000000000000000000000000000', 'in-id-0', 0, '\x'::bytea, '\x'::bytea, 'asset-0', 'asset0', 'def-0'),
			('0100000000000000000000000000000000000000000000000000000000000000', 'in-id-0', 1, '\x'::bytea, '\x'::bytea, 'asset-1', 'asset1', 'def-1'),
			('0200000000000000000000000000000000000000000000000000000000000000', 'in-id-1', 2, '\x'::bytea, '\x'::bytea, 'asset-2', 'asset2', 'def-2'),
			('0300000000000000000000000000000000000000000000000000000000000000', 'in-id-0', 3, '\x'::bytea, '\x'::bytea, 'asset-3', 'asset3', 'def-3');

		INSERT INTO manager_nodes (id, project_id, key_index, label) VALUES
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0'),
			('manager-node-id-1', 'proj-id-0', 1, 'manager-node-1');

		INSERT INTO accounts (id, manager_node_id, key_index, label) VALUES
			('account-id-0', 'manager-node-id-0', 0, 'account-0'),
			('account-id-1', 'manager-node-id-0', 1, 'account-1'),
			('account-id-2', 'manager-node-id-1', 2, 'account-2'),
			('account-id-3', 'manager-node-id-0', 3, 'account-3'),
			('account-id-4', 'manager-node-id-0', 4, 'account-4');
	`
)

func TestCreateProject(t *testing.T) {
	withContext(t, projectsFixtures, func(ctx context.Context) {
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
	withContext(t, projectsFixtures, func(ctx context.Context) {
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
	withContext(t, projectsFixtures, func(ctx context.Context) {
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
	withContext(t, projectsFixtures, func(ctx context.Context) {
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
				_ = pg.FromContext(ctx).QueryRow(ctx, q, ex.id).Scan(&got)
				if got != "new-name" {
					t.Errorf("name got=%v want new-name", got)
				}
			}
		}
	})
}

func TestArchiveProject(t *testing.T) {
	withContext(t, projectsFixtures+projectChildrenFixtures, func(ctx context.Context) {
		examples := []struct {
			id      string
			wantErr error
		}{
			{"proj-id-0", nil},
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
	})
}

func TestListMembers(t *testing.T) {
	withContext(t, projectsFixtures, func(ctx context.Context) {
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
	withContext(t, projectsFixtures, func(ctx context.Context) {
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
	withContext(t, projectsFixtures, func(ctx context.Context) {
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
	withContext(t, projectsFixtures, func(ctx context.Context) {
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
	err := pg.FromContext(ctx).QueryRow(ctx, q, projID, userID).Scan(&role)
	return role, err
}
