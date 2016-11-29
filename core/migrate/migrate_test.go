package migrate

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"chain-stealth/database/pg/pgtest"
)

func TestLoadStatus(t *testing.T) {
	const migrationTable = `
		CREATE TABLE IF NOT EXISTS migrations (
			filename text NOT NULL,
			hash text NOT NULL,
			applied_at timestamp with time zone DEFAULT now() NOT NULL,
			PRIMARY KEY(filename)
		);
	`
	const oneMigration = `
		INSERT INTO migrations (filename, hash, applied_at)
		VALUES ('x', 'b4e0497804e46e0a0b0b8c31975b062152d551bac49c3c2e80932567b4085dcd', '2016-02-09T23:21:55 US/Pacific');
	`

	loc, err := time.LoadLocation("US/Pacific")
	if err != nil {
		t.Fatal(err)
	}
	oneMigrationTime := time.Date(2016, 2, 9, 23, 21, 55, 0, loc)

	cases := []struct {
		initSQL string
		migs    []migration
		want    []migration
	}{
		{
			initSQL: `SELECT 1;`,
			migs:    nil,
			want:    nil,
		},
		{
			initSQL: migrationTable,
			migs:    nil,
			want:    nil,
		},
		{
			initSQL: `SELECT 1;`,
			migs:    []migration{{Name: "x", SQL: "a"}},
			want:    []migration{{Name: "x", SQL: "a"}},
		},
		{
			initSQL: migrationTable + oneMigration,
			migs:    []migration{{Name: "x", SQL: "a"}},
			want:    []migration{{Name: "x", SQL: "a", AppliedAt: oneMigrationTime}},
		},
		{
			initSQL: `SELECT 1;`,
			migs: []migration{
				{Name: "x", SQL: "a"},
				{Name: "y", SQL: "b"},
			},
			want: []migration{
				{Name: "x", SQL: "a"},
				{Name: "y", SQL: "b"},
			},
		},
		{
			initSQL: migrationTable + oneMigration,
			migs: []migration{
				{Name: "x", SQL: "a"},
				{Name: "y", SQL: "b"},
			},
			want: []migration{
				{Name: "x", SQL: "a", AppliedAt: oneMigrationTime},
				{Name: "y", SQL: "b"},
			},
		},
	}

	for testNum, test := range cases {
		_, db := pgtest.NewDB(t, "testdata/empty.sql")

		got := make([]migration, len(test.migs))
		copy(got, test.migs)

		err := loadStatus(db, got)
		if err != nil {
			t.Error(err)
			continue
		}

		// Comparing structs with time.Time using reflect.DeepEqual is a pain
		// because of the *time.Location, so we manually iterate and compare.
		if len(got) != len(test.want) {
			t.Errorf("%d: got=%#v want=%#v", testNum, got, test.want)
			continue
		}
		for i, got := range got {
			want := test.want[i]
			if got.Name != want.Name || got.Hash != want.Hash {
				t.Errorf("%d: migration %v, got=%#v want=%#v", testNum, i, got, want)
			}
		}
	}
}

func TestApplyMigrationSQL(t *testing.T) {
	save := migrations
	defer func() { migrations = save }()

	_, db := pgtest.NewDB(t, "testdata/empty.sql")

	migrations = []migration{{
		Name: "test-migration",
		SQL:  `CREATE TABLE test_table (a int);`,
	}}

	h := sha256.Sum256([]byte(migrations[0].SQL))
	migrations[0].Hash = hex.EncodeToString(h[:])
	err := Run(db)
	if err != nil {
		t.Error(err)
	}
}
