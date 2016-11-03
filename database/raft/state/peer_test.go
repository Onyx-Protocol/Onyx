package state

import (
	"reflect"
	"testing"
)

func TestRemovePeerAddr(t *testing.T) {
	s := State{peers: map[uint64]string{1: "1.2.3.4:567"}}
	expected_s := State{peers: map[uint64]string{}}

	s.RemovePeerAddr(1)
	if !reflect.DeepEqual(s, expected_s) {
		t.Errorf("RemovePeerAddr(%d) => %v want %v", 1, s, expected_s)
	}

}

func TestSetPeerAddr(*testing.T) {
}

func TestGetPeerAddr(*testing.T) {
}
