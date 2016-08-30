package pgtest

import (
	"net/url"
	"os/exec"
	"strings"
	"testing"
)

// Dump performs a full pg_dump of the data in the database at the
// provided URL.
func Dump(t testing.TB, dbURL string, includeSchema bool, excludingTables ...string) string {
	u, err := url.Parse(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimLeft(u.Path, "/")

	args := []string{"--no-owner", "--no-privileges", "--inserts"}
	if !includeSchema {
		args = append(args, "--data-only")
	}
	for _, tbl := range excludingTables {
		args = append(args, "--exclude-table="+tbl)
	}
	args = append(args, name)

	cmd := exec.Command("pg_dump", args...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
