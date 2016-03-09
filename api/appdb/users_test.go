package appdb_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"

	. "chain/api/appdb"
	"chain/api/asset/assettest"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/authn"
	"chain/testutil"
)

func init() {
	SetPasswordBCryptCost(bcrypt.MinCost)
}

func getUserByCreds(ctx context.Context, email, password string) (*User, error) {
	q := `SELECT id, password_hash FROM users WHERE lower(email) = lower($1)`
	var (
		id    string
		phash []byte
	)
	err := pg.QueryRow(ctx, q, email).Scan(&id, &phash)
	if err != nil {
		return nil, errors.Wrap(err, "user lookup")
	}

	if bcrypt.CompareHashAndPassword(phash, []byte(password)) != nil {
		return nil, errors.New("password does not match")
	}

	return &User{id, email}, nil
}

func TestCreateUser(t *testing.T) {
	ctx := pgtest.NewContext(t, "")
	defer pgtest.Finish(ctx)

	u1, err := CreateUser(ctx, "foo@bar.com", "abracadabra")
	if err != nil {
		t.Fatal(err)
	}

	if u1.Email != "foo@bar.com" {
		t.Errorf("email = %v want foo@bar.com", u1.Email)
	}

	if len(u1.ID) == 0 {
		t.Errorf("user ID length is zero")
	}

	// Fetch newly-created user
	u2, err := getUserByCreds(ctx, "foo@bar.com", "abracadabra")
	if err != nil {
		t.Fatal(err)
	}

	if *u2 != *u1 {
		t.Errorf("getUserByCreds = %v want %v", u2, u1)
	}

	// Try the same, but with a bad password
	_, err = getUserByCreds(ctx, "foo@bar.com", "arbadacarba")
	if err == nil {
		t.Error("getUserByCreds err should not nil")
	}
}

func TestCreateUserPreserveCase(t *testing.T) {
	ctx := pgtest.NewContext(t, "")
	defer pgtest.Finish(ctx)

	u1, err := CreateUser(ctx, "Foo@Bar.com", "abracadabra")
	if err != nil {
		t.Fatal(err)
	}

	if u1.Email != "Foo@Bar.com" {
		t.Errorf("email = %v want Foo@Bar.com", u1.Email)
	}

	// Ensure that case was preserved
	var (
		email string
		q     = "SELECT email FROM users where lower(email) = 'foo@bar.com'"
	)
	err = pg.QueryRow(ctx, q).Scan(&email)
	if err != nil {
		t.Fatal(err)
	}

	if email != "Foo@Bar.com" {
		t.Errorf("case not preserved, got = %v want Foo@Bar.com", email)
	}
}

func TestCreateUserNoDupes(t *testing.T) {
	examples := []struct{ email0, email1 string }{
		{"foo@bar.com", "foo@bar.com"},
		{"foo@bar.com", "Foo@bar.com"},     // test case-insensitivity
		{"foo@bar.com", "  foo@bar.com  "}, // test whitespace-insensitivity
	}

	for _, ex := range examples {
		t.Log("Example", ex.email0, ex.email1)

		func() {
			ctx := pgtest.NewContext(t, "")
			defer pgtest.Finish(ctx)

			_, err := CreateUser(ctx, ex.email0, "abracadabra")
			if err != nil {
				t.Fatal(err)
			}

			_, err = CreateUser(ctx, ex.email1, "abracadabra")
			if errors.Root(err) != ErrUserAlreadyExists {
				t.Errorf("error = %v want %v", err, ErrUserAlreadyExists)
			}
		}()
	}
}

func TestCreateUserInvalid(t *testing.T) {
	cases := []struct{ email, password string }{
		// missing password
		{"foo@bar.com", ""},
		// password too short
		{"foo@bar.com", "12345"},
		// password too long
		{"foo@bar.com", "123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890"},
		// missing email
		{"", "abracadabra"},
		// email too long
		{"123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890@bar.com", "abracadabra"},
	}

	for _, test := range cases {
		func() {
			ctx := pgtest.NewContext(t)
			defer pgtest.Finish(ctx)

			_, err := CreateUser(ctx, test.email, test.password)
			if err == nil {
				t.Errorf("CreateUser(%q, %q) err = nil want error", test.email, test.password)
			}
		}()
	}
}

func TestAuthenticateUserCreds(t *testing.T) {
	ctx := pgtest.NewContext(t, "")
	defer pgtest.Finish(ctx)

	u, err := CreateUser(ctx, "foo@bar.com", "abracadabra")
	if err != nil {
		t.Fatal(err)
	}

	gotID, err := AuthenticateUserCreds(ctx, "foo@bar.com", "abracadabra")
	if err != nil {
		t.Errorf("valid auth err = %v expected nil", err)
	}
	if gotID != u.ID {
		t.Errorf("got user ID = %v want %v", gotID, u.ID)
	}

	// Capitalization shouldn't matter
	gotID, err = AuthenticateUserCreds(ctx, "Foo@Bar.com", "abracadabra")
	if err != nil {
		t.Errorf("case-insensitive auth err = %v expected nil", err)
	}
	if gotID != u.ID {
		t.Errorf("got user ID = %v want %v", gotID, u.ID)
	}

	// Whitespace in the email shouldn't matter
	gotID, err = AuthenticateUserCreds(ctx, "  foo@bar.com  ", "abracadabra")
	if err != nil {
		t.Errorf("whitespace auth err = %v expected nil", err)
	}
	if gotID != u.ID {
		t.Errorf("got user ID = %v want %v", gotID, u.ID)
	}

	// Invalid email should yield error
	_, err = AuthenticateUserCreds(ctx, "nonexistent@bar.com", "abracadabra")
	if err != authn.ErrNotAuthenticated {
		t.Errorf("bad email auth error got = %v want %v", err, authn.ErrNotAuthenticated)
	}

	// Invalid password should yield error
	_, err = AuthenticateUserCreds(ctx, "foo@bar.com", "bad-password")
	if err != authn.ErrNotAuthenticated {
		t.Errorf("bad password auth error got = %v want %v", err, authn.ErrNotAuthenticated)
	}
}

func TestGetUser(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	id := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra")

	examples := []struct {
		id       string
		wantUser *User
		wantErr  error
	}{
		{id, &User{ID: id, Email: "foo@bar.com"}, nil},
		{"nonexistent", nil, pg.ErrUserInputNotFound},
	}

	for _, ex := range examples {
		t.Log("id:", ex.id)

		gotUser, gotErr := GetUser(ctx, ex.id)

		if !reflect.DeepEqual(gotUser, ex.wantUser) {
			t.Errorf("user:\ngot:  %v\nwant: %v", gotUser, ex.wantUser)
		}

		if errors.Root(gotErr) != ex.wantErr {
			t.Errorf("error:\ngot:  %v\nwant: %v", gotErr, ex.wantErr)
		}
	}
}

func TestGetUserByEmail(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	id := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra")

	examples := []struct {
		email    string
		wantUser *User
		wantErr  error
	}{
		{"foo@bar.com", &User{ID: id, Email: "foo@bar.com"}, nil},
		{"Foo@Bar.com", &User{ID: id, Email: "foo@bar.com"}, nil},
		{"  foo@bar.com  ", &User{ID: id, Email: "foo@bar.com"}, nil},
		{"baz@bar.com", nil, pg.ErrUserInputNotFound},
	}

	for _, ex := range examples {
		t.Log("email:", ex.email)

		gotUser, gotErr := GetUserByEmail(ctx, ex.email)

		if !reflect.DeepEqual(gotUser, ex.wantUser) {
			t.Errorf("user:\ngot:  %v\nwant: %v", gotUser, ex.wantUser)
		}

		if errors.Root(gotErr) != ex.wantErr {
			t.Errorf("error:\ngot:  %v\nwant: %v", gotErr, ex.wantErr)
		}
	}
}

func TestUpdateUserEmail(t *testing.T) {
	examples := []struct {
		password string
		email    string
		want     error
	}{
		{"abracadabra", "bar@foo.com", nil},
		{"abracadabra", "foo@bar.com", nil},           // reset to same email
		{"abracadabra", "Foo@Bar.com", nil},           // reset to same email, modulo case
		{"abracadabra", "  foo@bar.com  ", nil},       // reset to same email, stripping whitespace
		{"abracadabra", "invalid-email", ErrBadEmail}, // new email is not valid
		{"abracadabra", "foo2@bar.com", ErrBadEmail},  // new email is already taken
		{"bad-password", "foo@bar.com", ErrPasswordCheck},
	}

	for i, ex := range examples {
		func() {
			t.Log("Example", i)

			ctx := pgtest.NewContext(t)
			defer pgtest.Finish(ctx)

			id1 := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra")
			assettest.CreateUserFixture(ctx, t, "foo2@bar.com", "abracadabra")

			err := UpdateUserEmail(ctx, id1, ex.password, ex.email)
			if errors.Root(err) != ex.want {
				t.Errorf("error = %v want %v", errors.Root(err), ex.want)
			}

			if ex.want == nil {
				_, err := getUserByCreds(ctx, strings.TrimSpace(ex.email), ex.password)
				if err != nil {
					t.Errorf("error = %v want nil", err)
				}
			}
		}()
	}
}

func TestUpdateUserPassword(t *testing.T) {
	examples := []struct {
		password string
		newpass  string
		want     error
	}{
		{"abracadabra", "opensesame", nil},
		{"abracadabra", "", ErrBadPassword},
		{"bad-password", "opensesame", ErrPasswordCheck},
	}

	for i, ex := range examples {
		func() {
			t.Log("Example", i)

			ctx := pgtest.NewContext(t)
			defer pgtest.Finish(ctx)

			id1 := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra")

			err := UpdateUserPassword(ctx, id1, ex.password, ex.newpass)
			if errors.Root(err) != ex.want {
				t.Errorf("error = %v want %v", errors.Root(err), ex.want)
			}

			if ex.want == nil {
				_, err := getUserByCreds(ctx, "foo@bar.com", ex.newpass)
				if err != nil {
					t.Errorf("error = %v want nil", err)
				}
			}
		}()
	}
}

func TestPasswordResetFlow(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	id1 := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra")

	secret, err := StartPasswordReset(ctx, "foo@bar.com", time.Now())
	if err != nil {
		t.Fatal("unexepcted error:", err)
	}

	err = FinishPasswordReset(ctx, "foo@bar.com", secret, "open-sesame")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	err = CheckPassword(ctx, id1, "open-sesame")
	if err != nil {
		t.Errorf("check password error got = %v want nil", err)
	}

	// The reset secret should have been invalidated.
	err = FinishPasswordReset(ctx, "foo@bar.com", secret, "should-not-work")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Errorf("reset password error got = %v want %v", errors.Root(err), pg.ErrUserInputNotFound)
	}

	// The second attempt should not have changed anything.
	err = CheckPassword(ctx, id1, "open-sesame")
	if err != nil {
		t.Errorf("check password error got = %v want nil", err)
	}
}

func TestStartPasswordResetErrs(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		gotSecret, gotErr := StartPasswordReset(ctx, "foo@bar.com", time.Now())

		if gotSecret != "" {
			t.Errorf("secret got = %v want blank", gotSecret)
		}

		if errors.Root(gotErr) != ErrNoUserForEmail {
			t.Errorf("error got = %v want %v", errors.Root(gotErr), ErrNoUserForEmail)
		}
	})
}

func TestCheckPasswordReset(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	assettest.CreateUserFixture(ctx, t, "foo@bar.com", "anything")
	assettest.CreateUserFixture(ctx, t, "bar@foo.com", "anything")

	secret1, err := StartPasswordReset(ctx, "foo@bar.com", time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}
	secret2, err := StartPasswordReset(ctx, "bar@foo.com", time.Now().Add(PWResetLifeTime()*-2))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	examples := []struct {
		email  string
		secret string
		want   error
	}{
		// Valid example
		{"foo@bar.com", secret1, nil},
		// Valid example, mismatching email case
		{"Foo@Bar.com", secret1, nil},
		// Valid example, whitespace in email
		{"  foo@bar.com  ", secret1, nil},
		// Bad secret
		{"foo@bar.com", "bad-secret", pg.ErrUserInputNotFound},
		// Password reset has expired
		{"bar@foo.com", secret2, pg.ErrUserInputNotFound},
		// Bad user
		{"nonexistent", secret2, pg.ErrUserInputNotFound},
	}

	for _, ex := range examples {
		t.Logf("Example: %s:%s", ex.email, ex.secret)
		got := CheckPasswordReset(ctx, ex.email, ex.secret)
		if errors.Root(got) != ex.want {
			t.Errorf("error got = %v want %v", errors.Root(got), ex.want)
		}
	}
}

func TestFinishPasswordResetErrs(t *testing.T) {
	examples := []struct {
		email     string
		useSecret bool
		newpass   string
		resetTime time.Time
		wantErr   error
	}{
		// Valid example
		{"foo@bar.com", true, "new-password", time.Now(), nil},
		// Valid example, mismatching email case
		{"Foo@Bar.com", true, "new-password", time.Now(), nil},
		// Valid example, extra whitespace
		{"  foo@bar.com  ", true, "new-password", time.Now(), nil},
		// Invalid proposed password
		{"foo@bar.com", true, "", time.Now(), ErrBadPassword},
		// Bad secret
		{"foo@bar.com", false, "new-password", time.Now(), pg.ErrUserInputNotFound},
		// Password reset has expired
		{"foo@bar.com", true, "new-password", time.Now().Add(PWResetLifeTime() * -2), pg.ErrUserInputNotFound},
		// Bad user
		{"nonexistent", true, "new-password", time.Now(), pg.ErrUserInputNotFound},
	}

	for _, ex := range examples {
		t.Logf("Example: %s:%v", ex.email, ex.useSecret)

		func() {
			ctx := pgtest.NewContext(t)
			defer pgtest.Finish(ctx)

			assettest.CreateUserFixture(ctx, t, "foo@bar.com", "anything")
			secret1, err := StartPasswordReset(ctx, "foo@bar.com", ex.resetTime)
			if err != nil {
				testutil.FatalErr(t, err)
			}

			secret := "bad-secret"
			if ex.useSecret {
				secret = secret1
			}

			gotErr := FinishPasswordReset(ctx, ex.email, secret, ex.newpass)
			if errors.Root(gotErr) != ex.wantErr {
				t.Errorf("error got = %v want %v", errors.Root(gotErr), ex.wantErr)
			}
		}()
	}
}
