package state

import (
	"bytes"
	"context"

	"github.com/golang/protobuf/proto"

	"chain/database/raft/internal/statepb"
	"chain/errors"
	"chain/log"
)

const nextNodeID = "raft/nextNodeID"

var ErrAlreadyApplied = errors.New("entry already applied")

// State is a general-purpose data store designed to accumulate
// and apply replicated updates from a raft log.
// The zero value is an empty State ready to use.
type State struct {
	state        map[string][]byte
	peers        map[uint64]string // id -> addr
	appliedIndex uint64
	version      map[string]uint64 //key -> value index
}

// New returns a new State
func New() *State {
	return &State{
		state:   map[string][]byte{nextNodeID: []byte("2")},
		peers:   make(map[uint64]string),
		version: make(map[string]uint64),
	}
}

// SetPeerAddr sets the address for the given peer.
func (s *State) SetPeerAddr(id uint64, addr string) {
	s.peers[id] = addr
}

// GetPeerAddr gets the current address for the given peer, if set.
func (s *State) GetPeerAddr(id uint64) (addr string) {
	return s.peers[id]
}

// RemovePeerAddr deletes the current address for the given peer if it exists.
func (s *State) RemovePeerAddr(id uint64) {
	delete(s.peers, id)
}

// RestoreSnapshot decodes data and overwrites the contents of s.
// It should be called with the retrieved snapshot
// when bootstrapping a new node from an existing cluster
// or when recovering from a file on disk.
func (s *State) RestoreSnapshot(data []byte, index uint64) error {
	s.appliedIndex = index
	//TODO (ameets): think about having statepb in state for restore
	snapshot := &statepb.Snapshot{}
	err := proto.Unmarshal(data, snapshot)
	s.peers = snapshot.Peers
	s.state = snapshot.State //TODO (ameets): need to add version here
	log.Messagef(context.Background(), "decoded snapshot %#v (err %v)", s, err)
	return errors.Wrap(err)
}

// Snapshot returns an encoded copy of s
// suitable for RestoreSnapshot.
func (s *State) Snapshot() ([]byte, uint64, error) {
	log.Messagef(context.Background(), "encoding snapshot %#v", s)
	data, err := proto.Marshal(&statepb.Snapshot{
		State: s.state,
		Peers: s.peers,
	})
	return data, s.appliedIndex, errors.Wrap(err)
}

// Apply applies a raft log entry payload to s.
// For conditional operations returns whether codition was satisfied
// in addition to any errors.
func (s *State) Apply(data []byte, index uint64) (satisfied bool, err error) {
	if index < s.appliedIndex {
		return false, ErrAlreadyApplied
	}
	instr := &statepb.Instruction{}
	err = proto.Unmarshal(data, instr)
	if err != nil {
		// An error here indicates a malformed update
		// was written to the raft log. We do version
		// negotiation in the transport layer, so this
		// should be impossible; by this point, we are
		// all speaking the same version.
		return false, errors.Wrap(err)
	}

	log.Messagef(context.Background(), "state instruction: %v", instr)
	s.appliedIndex = index
	for _, cond := range instr.Conditions {
		y := true
		switch cond.Type {

		case statepb.Cond_NOT_KEY_EXISTS:
			y = false
			fallthrough
		case statepb.Cond_KEY_EXISTS:
			if _, ok := s.state[cond.Key]; ok != y {
				return false, nil
			}
		case statepb.Cond_NOT_VALUE_EQUAL:
			y = false
			fallthrough
		case statepb.Cond_VALUE_EQUAL:
			if ok := bytes.Equal(s.state[cond.Key], cond.Value); ok != y {
				return false, nil
			}
		case statepb.Cond_NOT_INDEX_EQUAL:
			y = false
			fallthrough
		case statepb.Cond_INDEX_EQUAL:
			if ok := (s.version[cond.Key] == cond.Index); ok != y {
				return false, nil
			}
		default:
			return false, errors.New("unknown condition type")
		}
	}
	for _, op := range instr.Operations {
		switch op.Type {
		case statepb.Op_SET:
			s.state[op.Key] = op.Value
			s.version[op.Key] = index
		case statepb.Op_DELETE:
			delete(s.state, op.Key)
			delete(s.version, op.Key)
			//TODO (ameets):increment version or delete entry?
			//s.version[op.Key] = index
		default:
			return false, errors.New("unknown operation type")
		}
	}

	return true, nil
}

// Provisional read operation.
func (s *State) Get(key string) (value []byte) {
	return s.state[key]
}

// Set encodes a set operation setting key to value.
// The encoded op should be committed to the raft log,
// then it can be applied with Apply.
func Set(key string, value []byte) (instruction []byte) {
	// TODO(kr): make a way to delete things
	b, _ := proto.Marshal(&statepb.Instruction{
		Operations: []*statepb.Op{{
			Type:  statepb.Op_SET,
			Key:   key,
			Value: value,
		}},
	})

	return b
}

// Insert encodes an insert operation. It is the same as Set,
// except it adds the condition that nothing can exist at the
// given key.
func Insert(key string, value []byte) (instruction []byte) {
	b, _ := proto.Marshal(&statepb.Instruction{
		Operations: []*statepb.Op{{
			Type:  statepb.Op_SET,
			Key:   key,
			Value: value,
		}},
		Conditions: []*statepb.Cond{{
			Type: statepb.Cond_NOT_KEY_EXISTS,
			Key:  key,
		}},
	})

	return b
}

// Delete encodes a delete operation for a given key.
// TODO (ameets):further commentary (?)
func Delete(key string) (instruction []byte) {
	b, _ := proto.Marshal(&statepb.Instruction{
		Operations: []*statepb.Op{{
			Type: statepb.Op_DELETE,
			Key:  key,
		}},
	})

	return b
}

// AppliedIndex returns the raft log index (applied index) of current state
func (s *State) AppliedIndex() uint64 {
	return s.appliedIndex
}

// IDCounter
func (s *State) NextNodeID() (id, version uint64) {
	id, n := proto.DecodeVarint(s.state[nextNodeID])
	if n == 0 {
		panic("raft: cannot decode nextNodeID")
	}
	return id, s.version[nextNodeID]
}

func IncrementNextNodeID(oldID uint64, index uint64) (instruction []byte) {
	b, _ := proto.Marshal(&statepb.Instruction{
		Conditions: []*statepb.Cond{{
			Type:  statepb.Cond_INDEX_EQUAL,
			Key:   nextNodeID,
			Index: index,
		}},
		Operations: []*statepb.Op{{
			Type:  statepb.Op_SET,
			Key:   nextNodeID,
			Value: proto.EncodeVarint(oldID + 1),
		}},
	})

	return b
}
