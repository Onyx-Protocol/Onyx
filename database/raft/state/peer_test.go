package state

import (
	"reflect"
	"testing"
)

func TestRemovePeerAddr(t *testing.T) {
	s := State{peers: map[uint64]string{1: "1.2.3.4:567"}}
	want := State{peers: map[uint64]string{}}

	s.RemovePeerAddr(1)
	if !reflect.DeepEqual(s, want) {
		t.Errorf("RemovePeerAddr(%d) => %v want %v", 1, s, want)
	}
}

func TestSetPeerAddr(t *testing.T) {
	s := New()
	want := &State{
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
	s := New()
	s.SetPeerAddr(1, "1.2.3.4:567")
	want := "1.2.3.4:567"

	got := s.GetPeerAddr(1)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("s.GetPeerAddr(1) = %s, want %s", got, want)
	}
}
