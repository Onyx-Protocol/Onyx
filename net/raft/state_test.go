package raft

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"os"
	"sync"
)

func newTestState() *state {
	return &state{
		NodeIDCounter: 2,
		PeersByID:     make(map[uint64]string),
		Data:          make(map[string]string),
	}
}

// state provides a simple implementation of the State interface so that
// internal tests within this package can create and destroy clusters. It
// implements a really primitive kv store. Instructions and snapshots are
// encoded using the stdlib gob package. A snapshot is just a gob-encoded
// state struct.
type state struct {
	mu            sync.Mutex
	NodeIDCounter uint64
	Index         uint64
	PeersByID     map[uint64]string
	Data          map[string]string
}

func (s *state) AppliedIndex() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Index
}

func (s *state) SetAppliedIndex(index uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Index = index
}

func (s *state) SetPeerAddr(id uint64, addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PeersByID[id] = addr
}

func (s *state) RemovePeerAddr(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.PeersByID, id)
}

func (s *state) Peers() map[uint64]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.PeersByID
}

func (s *state) IsAllowedMember(addr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.Data["/allowed/"+addr]
	return ok
}

func (s *state) EmptyWrite() []byte {
	return encodeInstruction(instruction{})
}

func (s *state) ReadFile(filename string) (data []byte, err error) {
	return ioutil.ReadFile(filename)
}

func (s *state) WriteFile(name string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(name, data, perm)
}

func (s *state) NextNodeID() (id, version uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.NodeIDCounter, s.Index
}

func (s *state) IncrementNextNodeID(oldID uint64, index uint64) []byte {
	return encodeInstruction(instruction{
		RequireIndex:  index,
		SetNextNodeID: oldID + 1,
	})
}

func (s *state) Apply(data []byte, index uint64) (satisfied bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var inst instruction
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&inst)
	if err != nil {
		panic(err)
	}

	if inst.RequireIndex != 0 && inst.RequireIndex != s.Index {
		return false
	}

	s.Index = index
	if inst.SetNextNodeID != 0 {
		s.NodeIDCounter = inst.SetNextNodeID
	}
	for k, v := range inst.Set {
		s.Data[k] = v
	}
	return true
}

func (s *state) Snapshot() (data []byte, index uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var buf bytes.Buffer
	err = gob.NewEncoder(&buf).Encode(s)
	return buf.Bytes(), s.Index, err
}

func (s *state) RestoreSnapshot(data []byte, index uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(s)
	return err
}

func set(k, v string) []byte {
	return encodeInstruction(instruction{
		Set: map[string]string{k: v},
	})
}

func encodeInstruction(instr instruction) []byte {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(instr)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

type instruction struct {
	RequireIndex  uint64
	SetNextNodeID uint64
	Set           map[string]string
}
