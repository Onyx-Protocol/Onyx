package appdb

import (
	"database/sql"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

// A common fixture for use in other test files within the package.
const sampleAppFixture = `
	INSERT INTO projects (id, name) VALUES ('app-id-0', 'app-0');
`

// A fixture for tests within this file.
const applicationsFixtures = `
	INSERT INTO users (id, email, password_hash) VALUES
		('user-id-0', 'foo@bar.com', 'password-does-not-matter'),
		('user-id-1', 'baz@bar.com', 'password-does-not-matter'),
		('user-id-2', 'biz@bar.com', 'password-does-not-matter');

	INSERT INTO projects (id, name) VALUES
		('app-id-0', 'app-0'),
		('app-id-1', 'app-1');

	INSERT INTO members (project_id, user_id, role) VALUES
		('app-id-0', 'user-id-0', 'admin'),
		('app-id-1', 'user-id-0', 'developer'),
		('app-id-0', 'user-id-1', 'developer');
`

func TestCreateApplication(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, applicationsFixtures)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	a, err := CreateApplication(ctx, "new-app", "user-id-0")
	if err != nil {
		t.Fatal(err)
	}

	if a.ID == "" {
		t.Error("app ID is blank")
	}

	if a.Name != "new-app" {
		t.Errorf("app name = %v want new-app", a.Name)
	}

	// Make sure the user was set as an admin.
	role, err := checkRole(ctx, a.ID, "user-id-0")
	if err != nil {
		t.Fatal(err)
	}

	if role != "admin" {
		t.Errorf("user role = %v want admin", role)
	}
}

func TestListApplications(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, applicationsFixtures)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		userID string
		want   []*Application
	}{
		{
			"user-id-0",
			[]*Application{
				{"app-id-0", "app-0"},
				{"app-id-1", "app-1"},
			},
		},
		{
			"user-id-1",
			[]*Application{
				{"app-id-0", "app-0"},
			},
		},
		{
			"user-id-2",
			nil,
		},
	}

	for _, ex := range examples {
		t.Log("user:", ex.userID)

		got, err := ListApplications(ctx, ex.userID)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("apps:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestGetApplication(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, applicationsFixtures)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		id      string
		wantApp *Application
		wantErr error
	}{
		{"app-id-0", &Application{ID: "app-id-0", Name: "app-0"}, nil},
		{"app-id-1", &Application{ID: "app-id-1", Name: "app-1"}, nil},
		{"nonexistent", nil, pg.ErrUserInputNotFound},
	}

	for _, ex := range examples {
		t.Log("application:", ex.id)

		gotApp, gotErr := GetApplication(ctx, ex.id)

		if !reflect.DeepEqual(gotApp, ex.wantApp) {
			t.Errorf("app:\ngot:  %v\nwant: %v", gotApp, ex.wantApp)
		}

		if errors.Root(gotErr) != ex.wantErr {
			t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
		}
	}
}

func TestUpdateApplication(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, applicationsFixtures)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		id      string
		wantErr error
	}{
		{"app-id-0", nil},
		{"nonexistent", pg.ErrUserInputNotFound},
	}

	for _, ex := range examples {
		t.Log("application:", ex.id)

		err := UpdateApplication(ctx, ex.id, "new-name")

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
}

func TestListMembers(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, applicationsFixtures)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		appID string
		want  []*Member
	}{
		{
			"app-id-0",
			[]*Member{
				{"user-id-1", "baz@bar.com", "developer"},
				{"user-id-0", "foo@bar.com", "admin"},
			},
		},
		{
			"app-id-1",
			[]*Member{
				{"user-id-0", "foo@bar.com", "developer"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("app:", ex.appID)

		got, err := ListMembers(ctx, ex.appID)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("members:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestAddMember(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, applicationsFixtures)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	err := AddMember(ctx, "app-id-0", "user-id-2", "developer")
	if err != nil {
		t.Fatal(err)
	}

	role, err := checkRole(ctx, "app-id-0", "user-id-2")
	if err != nil {
		t.Fatal(err)
	}

	if role != "developer" {
		t.Errorf("role = %v want developer", role)
	}

	// Repeated attempts result in error.
	err = AddMember(ctx, "app-id-0", "user-id-2", "developer")
	if errors.Root(err) != ErrAlreadyMember {
		t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), ErrAlreadyMember)
	}

	// Invalid roles result in error
	err = AddMember(ctx, "app-id-0", "user-id-3", "benevolent-dictator")
	if errors.Root(err) != ErrBadRole {
		t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), ErrBadRole)
	}
}

func TestUpdateMember(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, applicationsFixtures)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	err := UpdateMember(ctx, "app-id-0", "user-id-0", "developer")
	if err != nil {
		t.Fatal(err)
	}

	role, err := checkRole(ctx, "app-id-0", "user-id-0")
	if err != nil {
		t.Fatal(err)
	}

	if role != "developer" {
		t.Errorf("role = %v want developer", role)
	}

	// Updates for non-existing users result in error.
	err = UpdateMember(ctx, "app-id-0", "user-id-2", "developer")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), pg.ErrUserInputNotFound)
	}

	// Invalid roles result in error
	err = UpdateMember(ctx, "app-id-0", "user-id-0", "benevolent-dictator")
	if errors.Root(err) != ErrBadRole {
		t.Errorf("error:\ngot:  %v\nwant: %v", errors.Root(err), ErrBadRole)
	}
}

func TestRemoveMember(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, applicationsFixtures)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	err := RemoveMember(ctx, "app-id-0", "user-id-0")
	if err != nil {
		t.Fatal(err)
	}

	_, err = checkRole(ctx, "app-id-0", "user-id-0")
	if err != sql.ErrNoRows {
		t.Errorf("error = %v want %v", err, sql.ErrNoRows)
	}

	// Shouldn't affect other members
	role, err := checkRole(ctx, "app-id-0", "user-id-1")
	if err != nil {
		t.Fatal(err)
	}

	if role != "developer" {
		t.Errorf("user-1 role in app-0 = %v want developer", role)
	}

	// Shouldn't affect other apps
	role, err = checkRole(ctx, "app-id-1", "user-id-0")
	if err != nil {
		t.Fatal(err)
	}

	if role != "developer" {
		t.Errorf("user-0 role in app-1 = %v want developer", role)
	}
}

func checkRole(ctx context.Context, appID, userID string) (string, error) {
	var (
		q = `
			SELECT role
			FROM members
			WHERE project_id = $1 AND user_id = $2
		`
		role string
	)
	err := pg.FromContext(ctx).QueryRow(q, appID, userID).Scan(&role)
	return role, err
}
