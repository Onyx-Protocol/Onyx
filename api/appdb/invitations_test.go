package appdb_test

import (
	"database/sql"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	. "chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestCreateInvitation(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
	`)
	defer pgtest.Finish(ctx)

	inv, err := CreateInvitation(ctx, "proj-id-0", "foo@bar.com", "developer")
	if err != nil {
		t.Fatal(err)
	}

	if inv.ID == "" {
		t.Fatal("error: ID is blank")
	}

	if inv.ProjectName != "proj-0" {
		t.Errorf("proj name got = %v want proj-0", inv.ProjectName)
	}

	got, err := getTestInvitation(ctx, inv.ID)
	if err != nil {
		t.Fatal(err)
	}

	want := testInvitation{
		id:     inv.ID,
		projID: "proj-id-0",
		email:  "foo@bar.com",
		role:   "developer",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("invitation:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestCreateInvitationEmailWhitespace(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
	`)
	defer pgtest.Finish(ctx)

	inv, err := CreateInvitation(ctx, "proj-id-0", "  foo@bar.com  ", "developer")
	if err != nil {
		t.Fatal(err)
	}

	got, err := getTestInvitation(ctx, inv.ID)
	if err != nil {
		t.Fatal(err)
	}

	want := testInvitation{
		id:     inv.ID,
		projID: "proj-id-0",
		email:  "foo@bar.com",
		role:   "developer",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("invitation:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestCreateInvitationErrs(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');

		INSERT INTO users (id, password_hash, email) VALUES
			('user-id-1', '{}', 'bar@foo.com');

		INSERT INTO members (user_id, project_id, role) VALUES
			('user-id-1', 'proj-id-0', 'developer');
	`)
	defer pgtest.Finish(ctx)

	examples := []struct {
		projID, email, role string
		wantErr             error
	}{
		// Invalid email
		{"proj-id-0", "invalid-email", "developer", ErrBadEmail},
		// Invalid role
		{"proj-id-0", "foo@bar.com", "benevolent-dictator", ErrBadRole},
		// Email is already part of the application
		{"proj-id-0", "bar@foo.com", "admin", ErrAlreadyMember},
	}

	for i, ex := range examples {
		t.Log("Example", i)
		inv, err := CreateInvitation(ctx, ex.projID, ex.email, ex.role)

		if inv != nil {
			t.Errorf("invitation = %v want nil", inv)
		}

		if errors.Root(err) != ex.wantErr {
			t.Errorf("error = %v want %v", errors.Root(err), ex.wantErr)
		}
	}
}

func TestGetInvitation(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');

		INSERT INTO users (id, password_hash, email) VALUES
			('user-id-0', '{}', 'foo@bar.com'),
			('user-id-1', '{}', 'bar@foo.com');

		INSERT INTO invitations (id, project_id, email, role) VALUES (
			'inv-id-0',
			'proj-id-0',
			'foo@bar.com',
			'admin'
		), (
			'inv-id-1',
			'proj-id-0',
			'Bar@Foo.com',
			'developer'
		), (
			'inv-id-2',
			'proj-id-0',
			'no-account-yet@foo.com',
			'developer'
		);
	`)
	defer pgtest.Finish(ctx)

	examples := []struct {
		id      string
		want    *Invitation
		wantErr error
	}{
		// Invitation to existing user account
		{
			id: "inv-id-0",
			want: &Invitation{
				ID:          "inv-id-0",
				ProjectID:   "proj-id-0",
				ProjectName: "proj-0",
				Email:       "foo@bar.com",
				Role:        "admin",
				UserID:      "user-id-0",
			},
		},

		// Invitation to existing user account with mismatching email case
		{
			id: "inv-id-1",
			want: &Invitation{
				ID:          "inv-id-1",
				ProjectID:   "proj-id-0",
				ProjectName: "proj-0",
				Email:       "Bar@Foo.com",
				Role:        "developer",
				UserID:      "user-id-1",
			},
		},

		// Invitation to email address with no corresponding user account
		{
			id: "inv-id-2",
			want: &Invitation{
				ID:          "inv-id-2",
				ProjectID:   "proj-id-0",
				ProjectName: "proj-0",
				Email:       "no-account-yet@foo.com",
				Role:        "developer",
				UserID:      "",
			},
		},

		// Non-existent invitation
		{
			id:      "nonexistent",
			want:    nil,
			wantErr: pg.ErrUserInputNotFound,
		},
	}

	for _, ex := range examples {
		t.Logf("Example: %v", ex.id)

		got, gotErr := GetInvitation(ctx, ex.id)

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("invitation:\ngot:  %v\nwant: %v", got, ex.want)
		}

		if errors.Root(gotErr) != ex.wantErr {
			t.Errorf("error got = %v want %v", errors.Root(gotErr), ex.wantErr)
		}
	}
}

func TestCreateUserFromInvitation(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');

		INSERT INTO invitations (id, project_id, email, role) VALUES (
			'inv-id-0',
			'proj-id-0',
			'foo@bar.com',
			'admin'
		);
	`)
	defer pgtest.Finish(ctx)

	user, err := CreateUserFromInvitation(ctx, "inv-id-0", "password")
	if err != nil {
		t.Fatal(err)
	}

	if user.Email != "foo@bar.com" {
		t.Errorf("email = %v want foo@bar.com", user.Email)
	}

	role, err := checkRole(ctx, "proj-id-0", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	if role != "admin" {
		t.Errorf("role = %v want admin", role)
	}

	// Attempting to accept the invitation twice should yield an error
	_, err = CreateUserFromInvitation(ctx, "inv-id-0", "password")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Errorf("error = %v want %v", errors.Root(err), pg.ErrUserInputNotFound)
	}
}

func TestCreateUserFromInvitationErrs(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');

		INSERT INTO invitations (id, project_id, email, role) VALUES (
			'inv-id-0',
			'proj-id-0',
			'foo@bar.com',
			'admin'
		);

		INSERT INTO users (id, password_hash, email) VALUES
			('user-id-0', '{}', 'foo@bar.com');
	`)
	defer pgtest.Finish(ctx)

	examples := []struct {
		id       string
		password string
		wantErr  error
	}{
		// Invalid password
		{"inv-id-0", "badpw", ErrBadPassword},
		// Non-existent invite
		{"nonexistent", "password", pg.ErrUserInputNotFound},
		// Pre-existing user account
		{"inv-id-0", "password", ErrUserAlreadyExists},
	}

	for _, ex := range examples {
		t.Logf("Example %s:%s", ex.id, ex.password)

		user, err := CreateUserFromInvitation(ctx, ex.id, ex.password)

		if user != nil {
			t.Errorf("user should be nil")
		}

		if errors.Root(err) != ex.wantErr {
			t.Errorf("error = %v want %v", errors.Root(err), ex.wantErr)
		}
	}
}

func TestAddMemberFromInvitation(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');

		INSERT INTO users (id, password_hash, email) VALUES
			('user-id-0', '{}', 'foo@bar.com');

		INSERT INTO invitations (id, project_id, email, role) VALUES (
			'inv-id-0',
			'proj-id-0',
			'foo@bar.com',
			'admin'
		);
	`)
	defer pgtest.Finish(ctx)

	err := AddMemberFromInvitation(ctx, "inv-id-0")
	if err != nil {
		t.Fatal(err)
	}

	role, err := checkRole(ctx, "proj-id-0", "user-id-0")
	if err != nil {
		t.Fatal(err)
	}

	if role != "admin" {
		t.Errorf("role = %v want admin", role)
	}

	// Attempting to accept the invitation twice should yield an error
	err = AddMemberFromInvitation(ctx, "inv-id-0")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Errorf("error = %v want %v", errors.Root(err), pg.ErrUserInputNotFound)
	}
}

func TestAddMemberFromInvitationErrs(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');

		INSERT INTO invitations (id, project_id, email, role) VALUES (
			'inv-id-0',
			'proj-id-0',
			'foo@bar.com',
			'admin'
		);
	`)
	defer pgtest.Finish(ctx)

	examples := []struct {
		id      string
		wantErr error
	}{
		// Non-existent invite
		{"nonexistent", pg.ErrUserInputNotFound},
		// User doesn't exist
		{"inv-id-0", ErrInviteUserDoesNotExist},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.id)

		err := AddMemberFromInvitation(ctx, ex.id)
		if errors.Root(err) != ex.wantErr {
			t.Errorf("error = %v want %v", errors.Root(err), ex.wantErr)
		}
	}
}

func TestDeleteInvitation(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');

		INSERT INTO invitations (id, project_id, email, role) VALUES (
			'inv-id-0',
			'proj-id-0',
			'foo@bar.com',
			'admin'
		), (
			'inv-id-1',
			'proj-id-0',
			'Bar@Foo.com',
			'developer'
		);
	`)
	defer pgtest.Finish(ctx)

	err := DeleteInvitation(ctx, "inv-id-0")
	if err != nil {
		t.Fatal(err)
	}

	// The specified invitation should have been deleted...
	_, err = getTestInvitation(ctx, "inv-id-0")
	if errors.Root(err) != sql.ErrNoRows {
		t.Fatalf("error = %v want %v", errors.Root(err), sql.ErrNoRows)
	}

	// ...but other rows should not have been affected
	other, err := getTestInvitation(ctx, "inv-id-1")
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(other, testInvitation{}) {
		t.Error("other invitation is blank")
	}
}

type testInvitation struct {
	id     string
	projID string
	email  string
	role   string
}

func getTestInvitation(ctx context.Context, id string) (testInvitation, error) {
	var (
		q = `
			SELECT project_id, email, role
			FROM invitations
			WHERE id = $1
		`
		inv = testInvitation{id: id}
	)

	err := pg.FromContext(ctx).QueryRow(ctx, q, id).Scan(
		&inv.projID,
		&inv.email,
		&inv.role,
	)
	if err != nil {
		return testInvitation{}, errors.Wrap(err, "select query")
	}
	return inv, nil
}
