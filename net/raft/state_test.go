package raft

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"os"
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
	NodeIDCounter uint64
	Index         uint64
	PeersByID     map[uint64]string
	Data          map[string]string
}

func (s *state) AppliedIndex() uint64 {
	return s.Index
}

func (s *state) SetAppliedIndex(index uint64) {
	s.Index = index
}

func (s *state) SetPeerAddr(id uint64, addr string) {
	s.PeersByID[id] = addr
}

func (s *state) RemovePeerAddr(id uint64) {
	delete(s.PeersByID, id)
}

func (s *state) Peers() map[uint64]string {
	return s.PeersByID
}

func (s *state) IsAllowedMember(addr string) bool {
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
	return s.NodeIDCounter, s.Index
}

func (s *state) IncrementNextNodeID(oldID uint64, index uint64) []byte {
	return encodeInstruction(instruction{
		RequireIndex:  index,
		SetNextNodeID: oldID + 1,
	})
}

func (s *state) Apply(data []byte, index uint64) (satisfied bool) {
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
	var buf bytes.Buffer
	err = gob.NewEncoder(&buf).Encode(s)
	return buf.Bytes(), s.Index, err
}

func (s *state) RestoreSnapshot(data []byte, index uint64) error {
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
