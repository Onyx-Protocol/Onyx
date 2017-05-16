// Package sinkdb provides a strongly consistent key-value store.
package sinkdb

import (
	"context"
	"net/http"

	"github.com/golang/protobuf/proto"

	"chain/database/raft"
	"chain/database/sinkdb/internal/sinkpb"
)

// Open initializes the key-value store and returns a database handle.
func Open(laddr, dir, bootURL string, httpClient *http.Client, useTLS bool) (*DB, error) {
	state := newState()
	sv, err := raft.Start(laddr, dir, bootURL, httpClient, useTLS, state)
	if err != nil {
		return nil, err
	}
	db := &DB{state: state, raft: sv}
	return db, nil
}

// DB provides access to an opened kv store.
type DB struct {
	state *state
	raft  *raft.Service
}

// Exec executes the provided operations. If all of the provided conditionals
// are met, all of the provided effects are applied atomically.
func (db *DB) Exec(ctx context.Context, ops ...Op) error {
	instr := new(sinkpb.Instruction)
	for _, op := range ops {
		if op.err != nil {
			return op.err
		}
		instr.Conditions = append(instr.Conditions, op.conds...)
		instr.Operations = append(instr.Operations, op.effects...)
	}
	encoded, err := proto.Marshal(instr)
	if err != nil {
		return err
	}
	return db.raft.Exec(ctx, encoded)
}

// Get performs a linearizable read of the provided key. The
// read value is unmarshalled into v.
func (db *DB) Get(ctx context.Context, key string, v proto.Message) (found bool, err error) {
	err = db.raft.RequestRead(ctx)
	if err != nil {
		return false, err
	}
	buf := db.state.get(key)
	if len(buf) == 0 {
		return false, err
	}
	return true, proto.Unmarshal(buf, v)
}

// GetInconsistent performs a non-linearizable read of the provided key.
// The value may be stale. The read value is unmarshalled into v.
func (db *DB) GetInconsistent(key string, v proto.Message) (found bool, err error) {
	buf := db.state.get(key) // read directly from state
	if len(buf) == 0 {
		return false, err
	}
	return true, proto.Unmarshal(buf, v)
}

// AddAllowedMember configures sinkdb to allow the provided address
// to participate in Raft.
func (db *DB) AddAllowedMember(ctx context.Context, addr string) error {
	instr, err := proto.Marshal(&sinkpb.Instruction{
		Operations: []*sinkpb.Op{{
			Key:   allowedMemberPrefix + "/" + addr,
			Value: []byte{0x01},
		}},
	})
	if err != nil {
		return err
	}
	return db.raft.Exec(ctx, instr)
}

// RaftService returns the raft service used for replication.
func (db *DB) RaftService() *raft.Service {
	return db.raft
}
