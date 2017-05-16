package sinkdb

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"testing"
)

func TestRemovePeerAddr(t *testing.T) {
	s := state{peers: map[uint64]string{1: "1.2.3.4:567"}}
	want := state{peers: map[uint64]string{}}

	s.RemovePeerAddr(1)
	if !reflect.DeepEqual(s, want) {
		t.Errorf("RemovePeerAddr(%d) => %v want %v", 1, s, want)
	}
}

func TestSetPeerAddr(t *testing.T) {
	s := newState()
	want := &state{
		state:   s.state,
		peers:   map[uint64]string{1: "1.2.3.4:567"},
		version: s.version,
	}

	s.SetPeerAddr(1, "1.2.3.4:567")
	if !reflect.DeepEqual(s, want) {
		t.Errorf("s.SetPeerAddr(1, \"1.2.3.4:567\") => %v, want %v", s, want)
	}
}

func TestGetPeerAddr(t *testing.T) {
	s := newState()
	s.SetPeerAddr(1, "1.2.3.4:567")
	want := "1.2.3.4:567"

	got := s.GetPeerAddr(1)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("s.GetPeerAddr(1) = %s, want %s", got, want)
	}
}

func TestAllowedMember(t *testing.T) {
	raftDir, err := ioutil.TempDir("", "sinkdb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(raftDir)

	sdb, err := Open("", raftDir, "", new(http.Client), false)
	if err != nil {
		t.Fatal(err)
	}

	err = sdb.AddAllowedMember(context.Background(), "1234")
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if !sdb.state.IsAllowedMember("1234") {
		t.Fatal("expected 1234 to be a potential member")
	}
	if sdb.state.IsAllowedMember("5678") {
		t.Fatal("expected 5678 to not be a potential member")
	}
}
