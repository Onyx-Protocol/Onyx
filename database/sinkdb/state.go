package sinkdb

import (
	"bytes"
	"log"
	"reflect"
	"sync"

	"github.com/golang/protobuf/proto"

	"chain/database/sinkdb/internal/sinkpb"
	"chain/errors"
)

const (
	nextNodeID          = "raft/nextNodeID"
	allowedMemberPrefix = "/raft/allowed"
	dbName              = "~/.chaincore/sinkdb"
)

// state is a general-purpose data store designed to accumulate
// and apply replicated updates from a raft log.
type state struct {
	mu           sync.Mutex
	state        map[string][]byte
	peers        map[uint64]string // id -> addr
	appliedIndex uint64
	version      map[string]uint64 //key -> value index

	store Store
}

// newState returns a new State.
func newState(db Store) *state {
	return &state{
		state:   map[string][]byte{nextNodeID: []byte("2")},
		peers:   make(map[uint64]string),
		version: make(map[string]uint64),
		store:   db,
	}
}

// SetAppliedIndex sets the applied index to the provided index.
func (s *state) SetAppliedIndex(index uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appliedIndex = index
}

// Peers returns the current set of peer nodes. The returned
// map must not be modified.
func (s *state) Peers() map[uint64]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.peers
}

// SetPeerAddr sets the address for the given peer.
func (s *state) SetPeerAddr(id uint64, addr string) {
	newPeers := make(map[uint64]string)
	s.mu.Lock()
	defer s.mu.Unlock()
	for nodeID, addr := range s.peers {
		newPeers[nodeID] = addr
	}
	newPeers[id] = addr
	s.peers = newPeers
}

// RemovePeerAddr deletes the current address for the given peer if it exists.
func (s *state) RemovePeerAddr(id uint64) {
	newPeers := make(map[uint64]string)
	s.mu.Lock()
	defer s.mu.Unlock()
	for nodeID, addr := range s.peers {
		if nodeID == id {
			continue
		}
		newPeers[nodeID] = addr
	}
	s.peers = newPeers
}

// RestoreSnapshot decodes data and overwrites the contents of s.
// It should be called with the retrieved snapshot
// when bootstrapping a new node from an existing cluster
// or when recovering from a file on disk.
func (s *state) RestoreSnapshot(data []byte, index uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.appliedIndex = index
	//TODO (ameets): think about having sinkpb in state for restore
	snapshot := &sinkpb.Snapshot{}
	err := proto.Unmarshal(data, snapshot)
	s.peers = snapshot.Peers
	s.state = snapshot.State
	s.version = snapshot.Version

	// Chain Core 1.2.x generates snapshots without populating versions,
	// so we manually initialize here to avoid nil errors after marshaling
	// and unmarshaling the snapshot, which turns an initialized-but-empty
	// map into a nil map
	if s.version == nil {
		s.version = make(map[string]uint64)
	}
	return errors.Wrap(err)
}

// Snapshot returns an encoded copy of s suitable for RestoreSnapshot.
func (s *state) Snapshot() ([]byte, uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := proto.Marshal(&sinkpb.Snapshot{
		Version: s.version,
		State:   s.state,
		Peers:   s.peers,
	})
	return data, s.appliedIndex, errors.Wrap(err)
}

// Apply applies a raft log entry payload to s. For conditional operations, it
// returns whether the condition was satisfied.
func (s *state) Apply(data []byte, index uint64) (satisfied bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < s.appliedIndex {
		panic(errors.New("entry already applied"))
	}
	instr := &sinkpb.Instruction{}
	err := proto.Unmarshal(data, instr)
	if err != nil {
		// An error here indicates a malformed update
		// was written to the raft log. We do version
		// negotiation in the transport layer, so this
		// should be impossible; by this point, we are
		// all speaking the same version.
		panic(err)
	}

	s.appliedIndex = index
	for _, cond := range instr.Conditions {
		y := true
		switch cond.Type {

		case sinkpb.Cond_NOT_KEY_EXISTS:
			y = false
			fallthrough
		case sinkpb.Cond_KEY_EXISTS:
			if _, ok := s.state[cond.Key]; ok != y {
				return false
			}
		case sinkpb.Cond_NOT_VALUE_EQUAL:
			y = false
			fallthrough
		case sinkpb.Cond_VALUE_EQUAL:
			if ok := bytes.Equal(s.state[cond.Key], cond.Value); ok != y {
				return false
			}
		case sinkpb.Cond_NOT_INDEX_EQUAL:
			y = false
			fallthrough
		case sinkpb.Cond_INDEX_EQUAL:
			if ok := (s.version[cond.Key] == cond.Index); ok != y {
				return false
			}
		default:
			panic(errors.New("unknown condition type"))
		}
	}
	for _, op := range instr.Operations {
		switch op.Type {
		case sinkpb.Op_SET:
			s.state[op.Key] = op.Value
			s.version[op.Key] = index
		case sinkpb.Op_DELETE:
			delete(s.state, op.Key)
			delete(s.version, op.Key)
		default:
			panic(errors.New("unknown operation type"))
		}
	}
	return true
}

// get performs a provisional read operation.
func (s *state) get(key string) ([]byte, Version) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.state[key]
	n := s.version[key]
	return b, Version{key, ok, n}
}

// AppliedIndex returns the raft log index (applied index) of current state
func (s *state) AppliedIndex() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.appliedIndex
}

// NextNodeID generates an ID for the next node to join the cluster.
func (s *state) NextNodeID() (id, version uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, n := proto.DecodeVarint(s.state[nextNodeID])
	if n == 0 {
		panic("raft: cannot decode nextNodeID")
	}
	return id, s.version[nextNodeID]
}

func (s *state) IsAllowedMember(addr string) bool {
	_, ver := s.get(allowedMemberPrefix + "/" + addr)
	return ver.Exists()
}

func (s *state) IncrementNextNodeID(oldID uint64, index uint64) (instruction []byte) {
	instruction, _ = proto.Marshal(&sinkpb.Instruction{
		Conditions: []*sinkpb.Cond{{
			Type:  sinkpb.Cond_INDEX_EQUAL,
			Key:   nextNodeID,
			Index: index,
		}},
		Operations: []*sinkpb.Op{{
			Type:  sinkpb.Op_SET,
			Key:   nextNodeID,
			Value: proto.EncodeVarint(oldID + 1),
		}},
	})
	return instruction
}

func (s *state) EmptyWrite() (instruction []byte) {
	instruction, _ = proto.Marshal(&sinkpb.Instruction{
		Operations: []*sinkpb.Op{{
			Type:  sinkpb.Op_SET,
			Key:   "/dummyWrite",
			Value: []byte(""),
		}}})
	return instruction
}

func (s *state) Write(name string, data []byte) error {
	log.Println(reflect.TypeOf(s.store))
	return s.store.Put(name, data)
}

func (s *state) Read(name string) ([]byte, error) {
	return s.store.Get(name)
}
