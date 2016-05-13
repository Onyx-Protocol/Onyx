package appdb_test

import (
	"database/sql"
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

func TestCreateInvitation(t *testing.T) {
	ctx := pgtest.NewContext(t)

	projID := assettest.CreateProjectFixture(ctx, t, "", "proj-0")
	inv, err := CreateInvitation(ctx, projID, "foo@bar.com", "developer")
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
		projID: projID,
		email:  "foo@bar.com",
		role:   "developer",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("invitation:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestCreateInvitationEmailWhitespace(t *testing.T) {
	ctx := pgtest.NewContext(t)

	projID := assettest.CreateProjectFixture(ctx, t, "", "proj-0")
	inv, err := CreateInvitation(ctx, projID, "  foo@bar.com  ", "developer")
	if err != nil {
		t.Fatal(err)
	}

	got, err := getTestInvitation(ctx, inv.ID)
	if err != nil {
		t.Fatal(err)
	}

	want := testInvitation{
		id:     inv.ID,
		projID: projID,
		email:  "foo@bar.com",
		role:   "developer",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("invitation:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestCreateInvitationErrs(t *testing.T) {
	ctx := pgtest.NewContext(t)

	userID := assettest.CreateUserFixture(ctx, t, "bar@foo.com", "")
	projID := assettest.CreateProjectFixture(ctx, t, userID, "proj-0")

	examples := []struct {
		projID, email, role string
		wantErr             error
	}{
		// Invalid email
		{projID, "invalid-email", "developer", ErrBadEmail},
		// Invalid role
		{projID, "foo@bar.com", "benevolent-dictator", ErrBadRole},
		// Email is already part of the application
		{projID, "bar@foo.com", "admin", ErrAlreadyMember},
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
	ctx := pgtest.NewContext(t)

	user0ID := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "")
	user1ID := assettest.CreateUserFixture(ctx, t, "bar@foo.com", "")
	projID := assettest.CreateProjectFixture(ctx, t, "", "proj-0")
	inv0ID := assettest.CreateInvitationFixture(ctx, t, projID, "foo@bar.com", "admin")
	inv1ID := assettest.CreateInvitationFixture(ctx, t, projID, "Bar@Foo.com", "developer")
	inv2ID := assettest.CreateInvitationFixture(ctx, t, projID, "no-account-yet@foo.com", "developer")

	examples := []struct {
		id      string
		want    *Invitation
		wantErr error
	}{
		// Invitation to existing user account
		{
			id: inv0ID,
			want: &Invitation{
				ID:          inv0ID,
				ProjectID:   projID,
				ProjectName: "proj-0",
				Email:       "foo@bar.com",
				Role:        "admin",
				UserID:      user0ID,
			},
		},

		// Invitation to existing user account with mismatching email case
		{
			id: inv1ID,
			want: &Invitation{
				ID:          inv1ID,
				ProjectID:   projID,
				ProjectName: "proj-0",
				Email:       "Bar@Foo.com",
				Role:        "developer",
				UserID:      user1ID,
			},
		},

		// Invitation to email address with no corresponding user account
		{
			id: inv2ID,
			want: &Invitation{
				ID:          inv2ID,
				ProjectID:   projID,
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
	ctx := startContextDBTx(t)

	projID := assettest.CreateProjectFixture(ctx, t, "", "proj-0")
	invID := assettest.CreateInvitationFixture(ctx, t, projID, "foo@bar.com", "admin")

	user, err := CreateUserFromInvitation(ctx, invID, "password")
	if err != nil {
		t.Fatal(err)
	}

	if user.Email != "foo@bar.com" {
		t.Errorf("email = %v want foo@bar.com", user.Email)
	}

	role, err := checkRole(ctx, projID, user.ID)
	if err != nil {
		t.Fatal(err)
	}

	if role != "admin" {
		t.Errorf("role = %v want admin", role)
	}

	// Attempting to accept the invitation twice should yield an error
	_, err = CreateUserFromInvitation(ctx, invID, "password")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Errorf("error = %v want %v", errors.Root(err), pg.ErrUserInputNotFound)
	}
}

func TestCreateUserFromInvitationErrs(t *testing.T) {
	ctx := pgtest.NewContext(t)
	projID := assettest.CreateProjectFixture(ctx, t, "", "proj-0")
	invID := assettest.CreateInvitationFixture(ctx, t, projID, "foo@bar.com", "admin")
	assettest.CreateUserFixture(ctx, t, "foo@bar.com", "")
	examples := []struct {
		id       string
		password string
		wantErr  error
	}{
		// Invalid password
		{invID, "badpw", ErrBadPassword},
		// Non-existent invite
		{"nonexistent", "password", pg.ErrUserInputNotFound},
		// Pre-existing user account
		{invID, "password", ErrUserAlreadyExists},
	}

	for _, ex := range examples {
		_, ctx, err := pg.Begin(ctx)
		if err != nil {
			testutil.FatalErr(t, err)
		}

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
	ctx := startContextDBTx(t)

	projID := assettest.CreateProjectFixture(ctx, t, "", "proj-0")
	userID := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "")
	invID := assettest.CreateInvitationFixture(ctx, t, projID, "foo@bar.com", "admin")

	err := AddMemberFromInvitation(ctx, invID)
	if err != nil {
		t.Fatal(err)
	}

	role, err := checkRole(ctx, projID, userID)
	if err != nil {
		t.Fatal(err)
	}

	if role != "admin" {
		t.Errorf("role = %v want admin", role)
	}

	// Attempting to accept the invitation twice should yield an error
	err = AddMemberFromInvitation(ctx, invID)
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Errorf("error = %v want %v", errors.Root(err), pg.ErrUserInputNotFound)
	}
}

func TestAddMemberFromInvitationErrs(t *testing.T) {
	ctx := pgtest.NewContext(t)

	projID := assettest.CreateProjectFixture(ctx, t, "", "proj-0")
	invID := assettest.CreateInvitationFixture(ctx, t, projID, "foo@bar.com", "admin")

	examples := []struct {
		id      string
		wantErr error
	}{
		// Non-existent invite
		{"nonexistent", pg.ErrUserInputNotFound},
		// User doesn't exist
		{invID, ErrInviteUserDoesNotExist},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.id)
		_, ctx, err := pg.Begin(ctx)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		err = AddMemberFromInvitation(ctx, ex.id)
		if errors.Root(err) != ex.wantErr {
			t.Errorf("error = %v want %v", errors.Root(err), ex.wantErr)
		}
	}
}

func TestDeleteInvitation(t *testing.T) {
	ctx := pgtest.NewContext(t)

	projID := assettest.CreateProjectFixture(ctx, t, "", "proj-0")
	inv0ID := assettest.CreateInvitationFixture(ctx, t, projID, "foo@bar.com", "admin")
	inv1ID := assettest.CreateInvitationFixture(ctx, t, projID, "Bar@Foo.com", "developer")

	err := DeleteInvitation(ctx, inv0ID)
	if err != nil {
		t.Fatal(err)
	}

	// The specified invitation should have been deleted...
	_, err = getTestInvitation(ctx, inv0ID)
	if errors.Root(err) != sql.ErrNoRows {
		t.Fatalf("error = %v want %v", errors.Root(err), sql.ErrNoRows)
	}

	// ...but other rows should not have been affected
	other, err := getTestInvitation(ctx, inv1ID)
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

	err := pg.QueryRow(ctx, q, id).Scan(
		&inv.projID,
		&inv.email,
		&inv.role,
	)
	if err != nil {
		return testInvitation{}, errors.Wrap(err, "select query")
	}
	return inv, nil
}
