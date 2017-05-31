package sinkdb

import (
	"github.com/golang/protobuf/proto"

	"chain/database/sinkdb/internal/sinkpb"
)

// Op represents a change to the data store.
// Each Op starts with conditions, boolean predicates over
// existing stored data.
// If all conditions return true, the Op is said to be satisfied.
// It then results in zero or more effects,
// mutations to apply to the data.
// If an Op is unsatisfied, it has no effect.
// The zero value of Op is a valid operation
// with no conditions (it is always satisfied)
// and no effects.
type Op struct {
	err     error
	conds   []*sinkpb.Cond
	effects []*sinkpb.Op
}

// All encodes the atomic application of all its arguments.
//
// The returned Op is satisfied if all arguments would be satisfied.
// Its effects (if satisfied) are the effects of the arguments.
func All(op ...Op) Op {
	var outer Op
	for _, inner := range op {
		if inner.err != nil {
			return inner
		}
		outer.conds = append(outer.conds, inner.conds...)
		outer.effects = append(outer.effects, inner.effects...)
	}
	return outer
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

// IfNotModified encodes a conditional to make an instruction
// successful only if the version stored in v's key matches v.
//
// If v.Exists() is false,
// IfNotModified(v) is equivalent to IfNotExists(v.Key()).
func IfNotModified(v Version) Op {
	return Op{
		conds: []*sinkpb.Cond{{
			Type:  sinkpb.Cond_INDEX_EQUAL,
			Key:   v.key,
			Index: v.n,
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

// AddAllowedMember configures sinkdb to allow the provided address
// to participate in Raft.
func AddAllowedMember(addr string) Op {
	return Op{
		effects: []*sinkpb.Op{{
			Key:   allowedMemberPrefix + "/" + addr,
			Value: []byte{0x01},
		}},
	}
}

// Error returns an Op representing an error condition.
// Exec will return err, and have no effect,
// when the returned Op is executed.
// If err is nil, Error returns the zero Op.
func Error(err error) Op {
	return Op{err: err}
}
