package main

import (
	"os"
	"path/filepath"
	"testing"
)

const testDir = "testfiles"

func TestRunMigration(t *testing.T) {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}
	files := []string{"select.go", "select.sql"}

	for _, f := range files {
		f = filepath.Join(testDir, f)
		err := runMigration(u, f)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
}
