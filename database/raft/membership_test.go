package raft

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestAllowedMember(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	raftDir := filepath.Join(currentDir, "/.testraft")
	err = os.Mkdir(raftDir, 0700)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(raftDir)

	raftDB, err := Start("", raftDir, "", new(http.Client), false)
	if err != nil {
		t.Fatal(err)
	}

	err = raftDB.AddAllowedMember(context.Background(), "1234")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if !raftDB.isAllowedMember(context.Background(), "1234") {
		t.Fatal("expected 1234 to be a potential member")
	}

	if raftDB.isAllowedMember(context.Background(), "5678") {
		t.Fatal("expected 5678 to not be a potential member")
	}
}
