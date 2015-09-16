package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/authn"
)

func getUserByCreds(ctx context.Context, email, password string) (*User, error) {
	q := `SELECT id, password_hash FROM users WHERE lower(email) = lower($1)`
	var (
		id    string
		phash []byte
	)
	err := pg.FromContext(ctx).QueryRow(q, email).Scan(&id, &phash)
	if err != nil {
		return nil, errors.Wrap(err, "user lookup")
	}

	if bcrypt.CompareHashAndPassword(phash, []byte(password)) != nil {
		return nil, errors.New("password does not match")
	}

	return &User{id, email}, nil
}

func TestCreateUser(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, "")
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

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
	dbtx := pgtest.TxWithSQL(t, "")
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

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
	err = dbtx.QueryRow(q).Scan(&email)
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
		{"foo@bar.com", "Foo@bar.com"}, // test case-insensitivity
	}

	for _, ex := range examples {
		t.Log("Example", ex.email0, ex.email1)

		func() {
			dbtx := pgtest.TxWithSQL(t, "")
			defer dbtx.Rollback()

			ctx := pg.NewContext(context.Background(), dbtx)

			_, err := CreateUser(ctx, ex.email0, "abracadabra")
			if err != nil {
				t.Fatal(err)
			}

			_, err = CreateUser(ctx, ex.email1, "abracadabra")
			if errors.Root(err) != ErrBadEmail {
				t.Errorf("error want = %v got %v", errors.Root(err), ErrBadEmail)
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
			dbtx := pgtest.TxWithSQL(t)
			defer dbtx.Rollback()
			ctx := pg.NewContext(context.Background(), dbtx)

			_, err := CreateUser(ctx, test.email, test.password)
			if err == nil {
				t.Errorf("CreateUser(%q, %q) err = nil want error", test.email, test.password)
			}
		}()
	}
}

func TestAuthenticateUserCreds(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, "")
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

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
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO users (id, email, password_hash)
		VALUES ('user-id-0', 'foo@bar.com', 'password-does-not-matter');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		id       string
		wantUser *User
		wantErr  error
	}{
		{"user-id-0", &User{ID: "user-id-0", Email: "foo@bar.com"}, nil},
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
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO users (id, email, password_hash)
		VALUES ('user-id-0', 'foo@bar.com', 'password-does-not-matter');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		email    string
		wantUser *User
		wantErr  error
	}{
		{"foo@bar.com", &User{ID: "user-id-0", Email: "foo@bar.com"}, nil},
		{"Foo@Bar.com", &User{ID: "user-id-0", Email: "foo@bar.com"}, nil},
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
	fix := `
		INSERT INTO users (id, email, password_hash)
		VALUES (
			'user-id-0',
			'foo@bar.com',
			'$2a$08$DNDEy/pOSfiiyW7o4qEzGO4Ae6gzQVtLVVFCTwO9cwWyekm/gFkxC'::bytea -- plaintext: abracadabra
		);

		INSERT INTO users (id, email, password_hash)
		VALUES (
			'user-id-1',
			'foo2@bar.com',
			'$2a$08$DNDEy/pOSfiiyW7o4qEzGO4Ae6gzQVtLVVFCTwO9cwWyekm/gFkxC'::bytea -- plaintext: abracadabra
		);
	`

	examples := []struct {
		password string
		email    string
		want     error
	}{
		{"abracadabra", "bar@foo.com", nil},
		{"abracadabra", "foo@bar.com", nil},           // reset to same email
		{"abracadabra", "Foo@Bar.com", nil},           // reset to same email, modulo case
		{"abracadabra", "invalid-email", ErrBadEmail}, // new email is not valid
		{"abracadabra", "foo2@bar.com", ErrBadEmail},  // new email is already taken
		{"bad-password", "foo@bar.com", ErrPasswordCheck},
	}

	for i, ex := range examples {
		func() {
			t.Log("Example", i)

			dbtx := pgtest.TxWithSQL(t, fix)
			defer dbtx.Rollback()
			ctx := pg.NewContext(context.Background(), dbtx)

			err := UpdateUserEmail(ctx, "user-id-0", ex.password, ex.email)
			if errors.Root(err) != ex.want {
				t.Errorf("error = %v want %v", errors.Root(err), ex.want)
			}

			if ex.want == nil {
				_, err := getUserByCreds(ctx, ex.email, ex.password)
				if err != nil {
					t.Errorf("error = %v want nil", err)
				}
			}
		}()
	}
}

func TestUpdateUserPassword(t *testing.T) {
	fix := `
		INSERT INTO users (id, email, password_hash)
		VALUES (
			'user-id-0',
			'foo@bar.com',
			'$2a$08$DNDEy/pOSfiiyW7o4qEzGO4Ae6gzQVtLVVFCTwO9cwWyekm/gFkxC'::bytea -- plaintext: abracadabra
		);
	`

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

			dbtx := pgtest.TxWithSQL(t, fix)
			defer dbtx.Rollback()
			ctx := pg.NewContext(context.Background(), dbtx)

			err := UpdateUserPassword(ctx, "user-id-0", ex.password, ex.newpass)
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
