package sinkdb

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestRestartDB(t *testing.T) {
	ctx := context.Background()

	raftDir, err := ioutil.TempDir("", "sinkdb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(raftDir)

	// Create a new fresh db and add an allowed member.
	sdb1, err := Open("", raftDir, new(http.Client))
	if err != nil {
		t.Fatal(err)
	}
	defer sdb1.Close()
	err = sdb1.RaftService().Init()
	if err != nil {
		t.Fatal(err)
	}
	err = sdb1.Exec(ctx, AddAllowedMember("1234"))
	if err != nil {
		t.Fatal(err)
	}
	err = sdb1.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Re-open the database and verify that the write is still there.
	sdb2, err := Open("", raftDir, new(http.Client))
	if err != nil {
		t.Fatal(err)
	}
	defer sdb2.Close()
	err = sdb2.RaftService().WaitRead(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !sdb2.state.IsAllowedMember("1234") {
		t.Error("expected allowed member to be persisted, but it wasn't")
	}
	err = sdb2.Close()
	if err != nil {
		t.Fatal(err)
	}
}
