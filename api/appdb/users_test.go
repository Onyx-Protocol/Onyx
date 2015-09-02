package appdb

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
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
		return nil, errors.New("password does not match") // TODO: should result in 401
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
			pqErr, ok := errors.Root(err).(*pq.Error)
			if !ok || pqErr.Code.Name() != "unique_violation" {
				t.Errorf("error = %v want unique_violation", err)
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
