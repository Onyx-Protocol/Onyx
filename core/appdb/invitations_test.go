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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	inv, err := CreateInvitation(ctx, "foo@bar.com", "developer")
	if err != nil {
		t.Fatal(err)
	}

	if inv.ID == "" {
		t.Fatal("error: ID is blank")
	}

	got, err := getTestInvitation(ctx, inv.ID)
	if err != nil {
		t.Fatal(err)
	}

	want := testInvitation{
		id:    inv.ID,
		email: "foo@bar.com",
		role:  "developer",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("invitation:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestCreateInvitationEmailWhitespace(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	inv, err := CreateInvitation(ctx, "  foo@bar.com  ", "developer")
	if err != nil {
		t.Fatal(err)
	}

	got, err := getTestInvitation(ctx, inv.ID)
	if err != nil {
		t.Fatal(err)
	}

	want := testInvitation{
		id:    inv.ID,
		email: "foo@bar.com",
		role:  "developer",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("invitation:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestCreateInvitationErrs(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	assettest.CreateUserFixture(ctx, t, "bar@foo.com", "", "")

	examples := []struct {
		email, role string
		wantErr     error
	}{
		// Invalid email
		{"invalid-email", "developer", ErrBadEmail},
		// Invalid role
		{"foo@bar.com", "benevolent-dictator", ErrBadRole},
		// Email is already part of the application
		{"bar@foo.com", "developer", ErrUserAlreadyExists},
	}

	for i, ex := range examples {
		t.Log("Example", i)
		inv, err := CreateInvitation(ctx, ex.email, ex.role)

		if inv != nil {
			t.Errorf("invitation = %v want nil", inv)
		}

		if errors.Root(err) != ex.wantErr {
			t.Errorf("error = %v want %v", errors.Root(err), ex.wantErr)
		}
	}
}

func TestGetInvitation(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	inv0ID := assettest.CreateInvitationFixture(ctx, t, "foo@bar.com", "developer")

	examples := []struct {
		id      string
		want    *Invitation
		wantErr error
	}{
		// Invitation to potential user
		{
			id: inv0ID,
			want: &Invitation{
				ID:    inv0ID,
				Email: "foo@bar.com",
				Role:  "developer",
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	invID := assettest.CreateInvitationFixture(ctx, t, "foo@bar.com", "admin")

	user, err := CreateUserFromInvitation(ctx, invID, "password")
	if err != nil {
		t.Fatal(err)
	}

	if user.Email != "foo@bar.com" {
		t.Errorf("email = %v want foo@bar.com", user.Email)
	}

	role, err := checkRole(ctx, user.ID)
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	invID := assettest.CreateInvitationFixture(ctx, t, "foo@bar.com", "developer")
	assettest.CreateUserFixture(ctx, t, "foo@bar.com", "", "")
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

func TestDeleteInvitation(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	inv0ID := assettest.CreateInvitationFixture(ctx, t, "foo@bar.com", "admin")
	inv1ID := assettest.CreateInvitationFixture(ctx, t, "Bar@Foo.com", "developer")

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
			SELECT email, role
			FROM invitations
			WHERE id = $1
		`
		inv = testInvitation{id: id}
	)

	err := pg.QueryRow(ctx, q, id).Scan(
		&inv.email,
		&inv.role,
	)
	if err != nil {
		return testInvitation{}, errors.Wrap(err, "select query")
	}
	return inv, nil
}
