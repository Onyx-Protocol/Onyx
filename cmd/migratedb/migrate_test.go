package main

import (
	"path/filepath"
	"testing"

	"chain/database/pg/pgtest"
)

const (
	testDir = "testfiles"
)

func testPath(filename string) string {
	return filepath.Join(testDir, filename)
}

func TestLoadMigrations(t *testing.T) {
	testCases := []struct {
		schemaFile   string
		migrationDir string
		want         []migration
	}{
		{
			schemaFile:   "empty.sql",
			migrationDir: "empty",
			want:         []migration{},
		},
		{
			schemaFile:   "migration-table.sql",
			migrationDir: "empty",
			want:         []migration{},
		},
		{
			schemaFile:   "empty.sql",
			migrationDir: "one-migration",
			want: []migration{
				{
					Filename: "select.sql",
					Hash:     "b4e0497804e46e0a0b0b8c31975b062152d551bac49c3c2e80932567b4085dcd",
				},
			},
		},
		{
			schemaFile:   "one-migration-applied.sql",
			migrationDir: "one-migration",
			want: []migration{
				{
					Filename: "select.sql",
					Hash:     "b4e0497804e46e0a0b0b8c31975b062152d551bac49c3c2e80932567b4085dcd",
					Applied:  true,
				},
			},
		},
		{
			schemaFile:   "empty.sql",
			migrationDir: "multiple",
			want: []migration{
				{
					Filename: "2015-11-03.0.api.example.sql",
					Hash:     "b4e0497804e46e0a0b0b8c31975b062152d551bac49c3c2e80932567b4085dcd",
				},
				{
					Filename: "2015-12-15.0.api.another-example.sql",
					Hash:     "a41109d24069b4822ddc5f367b25d484dc7e839bff338ce7a3e5da641caacda0",
				},
			},
		},
	}

	for testNum, tc := range testCases {
		_, db := pgtest.NewDB(t, testPath(tc.schemaFile))
		migrations, err := loadMigrations(db, testPath(tc.migrationDir))
		if err != nil {
			t.Error(err)
			continue
		}

		// Comparing structs with time.Time using reflect.DeepEqual is a pain
		// because of the *time.Location, so we manually iterate and compare.
		if len(migrations) != len(tc.want) {
			t.Errorf("%d: got=%#v want=%#v", testNum, migrations, tc.want)
			continue
		}
		for i, got := range migrations {
			want := tc.want[i]
			if got.Filename != want.Filename || got.Hash != want.Hash || got.Applied != want.Applied {
				t.Errorf("%d: migration %v, got=%#v want=%#v", testNum, i, got, want)
			}
		}
	}
}

func TestApplyMigrationSQL(t *testing.T) {
	url, db := pgtest.NewDB(t, "testfiles/empty.sql")
	migrations, err := loadMigrations(db, testPath("one-migration"))
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 1 {
		t.Errorf("len(migrations) = %d; want=1", len(migrations))
	}
	err = runMigration(db, url, testPath("one-migration"), migrations[0])
	if err != nil {
		t.Error(err)
	}
}

func TestApplyMigrationGo(t *testing.T) {
	url, db := pgtest.NewDB(t, "testfiles/empty.sql")
	migrations, err := loadMigrations(db, testPath("go-migration"))
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 1 {
		t.Errorf("len(migrations) = %d; want=1", len(migrations))
	}
	err = runMigration(db, url, testPath("go-migration"), migrations[0])
	if err != nil {
		t.Error(err)
	}
}
