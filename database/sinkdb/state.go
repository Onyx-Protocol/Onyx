package sinkdb

import (
	"bytes"

	"github.com/golang/protobuf/proto"

	"chain/database/sinkdb/internal/sinkpb"
	"chain/errors"
)

const (
	nextNodeID          = "raft/nextNodeID"
	allowedMemberPrefix = "/raft/allowed"
)

// state is a general-purpose data store designed to accumulate
// and apply replicated updates from a raft log.
type state struct {
	state        map[string][]byte
	peers        map[uint64]string // id -> addr
	appliedIndex uint64
	version      map[string]uint64 //key -> value index
}

// newState returns a new State.
func newState() *state {
	return &state{
		state:   map[string][]byte{nextNodeID: []byte("2")},
		peers:   make(map[uint64]string),
		version: make(map[string]uint64),
	}
}

// SetPeerAddr sets the address for the given peer.
func (s *state) SetPeerAddr(id uint64, addr string) {
	s.peers[id] = addr
}

// GetPeerAddr gets the current address for the given peer, if set.
func (s *state) GetPeerAddr(id uint64) (addr string) {
	return s.peers[id]
}

// RemovePeerAddr deletes the current address for the given peer if it exists.
func (s *state) RemovePeerAddr(id uint64) {
	delete(s.peers, id)
}

// RestoreSnapshot decodes data and overwrites the contents of s.
// It should be called with the retrieved snapshot
// when bootstrapping a new node from an existing cluster
// or when recovering from a file on disk.
func (s *state) RestoreSnapshot(data []byte, index uint64) error {
	s.appliedIndex = index
	//TODO (ameets): think about having sinkpb in state for restore
	snapshot := &sinkpb.Snapshot{}
	err := proto.Unmarshal(data, snapshot)
	s.peers = snapshot.Peers
	s.state = snapshot.State //TODO (ameets): need to add version here
	return errors.Wrap(err)
}

// Snapshot returns an encoded copy of s suitable for RestoreSnapshot.
func (s *state) Snapshot() ([]byte, uint64, error) {
	data, err := proto.Marshal(&sinkpb.Snapshot{
		State: s.state,
		Peers: s.peers,
	})
	return data, s.appliedIndex, errors.Wrap(err)
}

// Apply applies a raft log entry payload to s. For conditional operations, it
// returns whether the condition was satisfied.
func (s *state) Apply(data []byte, index uint64) (satisfied bool, err error) {
	if index < s.appliedIndex {
		return false, errors.New("entry already applied")
	}
	instr := &sinkpb.Instruction{}
	err = proto.Unmarshal(data, instr)
	if err != nil {
		// An error here indicates a malformed update
		// was written to the raft log. We do version
		// negotiation in the transport layer, so this
		// should be impossible; by this point, we are
		// all speaking the same version.
		return false, errors.Wrap(err)
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
				return false, nil
			}
		case sinkpb.Cond_NOT_VALUE_EQUAL:
			y = false
			fallthrough
		case sinkpb.Cond_VALUE_EQUAL:
			if ok := bytes.Equal(s.state[cond.Key], cond.Value); ok != y {
				return false, nil
			}
		case sinkpb.Cond_NOT_INDEX_EQUAL:
			y = false
			fallthrough
		case sinkpb.Cond_INDEX_EQUAL:
			if ok := (s.version[cond.Key] == cond.Index); ok != y {
				return false, nil
			}
		default:
			return false, errors.New("unknown condition type")
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
			return false, errors.New("unknown operation type")
		}
	}

	return true, nil
}

// get performs a provisional read operation.
func (s *state) get(key string) (value []byte) {
	return s.state[key]
}

// AppliedIndex returns the raft log index (applied index) of current state
func (s *state) AppliedIndex() uint64 {
	return s.appliedIndex
}

// NextNodeID generates an ID for the next node to join the cluster.
func (s *state) NextNodeID() (id, version uint64) {
	id, n := proto.DecodeVarint(s.state[nextNodeID])
	if n == 0 {
		panic("raft: cannot decode nextNodeID")
	}
	return id, s.version[nextNodeID]
}

func (s *state) IsAllowedMember(addr string) bool {
	data := s.get(allowedMemberPrefix + "/" + addr)
	return len(data) > 0
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
