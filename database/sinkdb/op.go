package sinkdb

import (
	"github.com/golang/protobuf/proto"

	"chain/database/sinkdb/internal/sinkpb"
)

type Op struct {
	err     error
	conds   []*sinkpb.Cond
	effects []*sinkpb.Op
}

// IfNotExists encodes a conditional to make an instruction
// successful only if the provided key does not exist.
func IfNotExists(key string) Op {
	return Op{
		conds: []*sinkpb.Cond{{
			Type: sinkpb.Cond_NOT_KEY_EXISTS,
			Key:  key,
		}},
	}
}

// Delete encodes a delete operation for key.
func Delete(key string) Op {
	return Op{
		effects: []*sinkpb.Op{{
			Type: sinkpb.Op_DELETE,
			Key:  key,
		}},
	}
}

// Set encodes a set operation setting key to value.
func Set(key string, value proto.Message) Op {
	encodedValue, err := proto.Marshal(value)
	if err != nil {
		return Op{err: err}
	}

	return Op{
		effects: []*sinkpb.Op{{
			Type:  sinkpb.Op_SET,
			Key:   key,
			Value: encodedValue,
		}},
	}
}
