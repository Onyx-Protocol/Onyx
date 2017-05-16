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
func IfNotExists(key string) (op Op) {
	op.conds = append(op.conds, &sinkpb.Cond{
		Type: sinkpb.Cond_NOT_KEY_EXISTS,
		Key:  key,
	})
	return op
}

// Delete encodes a delete operation for key.
func Delete(key string) (op Op) {
	op.effects = append(op.effects, &sinkpb.Op{
		Type: sinkpb.Op_DELETE,
		Key:  key,
	})
	return op
}

// Set encodes a set operation setting key to value.
func Set(key string, value proto.Message) (op Op) {
	encodedValue, err := proto.Marshal(value)
	if err != nil {
		op.err = err
		return op
	}

	op.effects = append(op.effects, &sinkpb.Op{
		Type:  sinkpb.Op_SET,
		Key:   key,
		Value: encodedValue,
	})
	return op
}
