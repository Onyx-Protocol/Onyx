package authz

import (
	"bytes"
	"context"
	"time"

	"github.com/golang/protobuf/proto"

	"chain/database/raft"
	"chain/errors"
)

// Generate code for the Grant and GrantList types.
//go:generate protoc -I. -I$CHAIN/.. --go_out=. grant.proto

// StoreGrant stores a new grant in the provided raft store
func StoreGrant(ctx context.Context, raftDB *raft.Service, grant Grant, grantPrefix string) (*Grant, error) {
	key := grantPrefix + grant.Policy
	if grant.CreatedAt == "" {
		grant.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := raftDB.Get(ctx, key)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	if data == nil {
		// if there aren't any grants associated with this policy, go ahead
		// and chuck this into raftdb
		gList := &GrantList{
			Grants: []*Grant{&grant},
		}
		val, err := proto.Marshal(gList)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		// TODO(tessr): Make this safe for concurrent updates. Will likely require a
		// conditional write operation for raftDB
		err = raftDB.Set(ctx, key, val)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		return &grant, nil
	}

	grantList := new(GrantList)
	err = proto.Unmarshal(data, grantList)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	grants := grantList.Grants
	for _, existing := range grants {
		if EqualGrants(*existing, grant) {
			// this grant already exists, return for idempotency
			return existing, nil
		}
	}

	// create new grant and it append to the list of grants associated with this policy
	grants = append(grants, &grant)
	gList := &GrantList{Grants: grants}
	val, err := proto.Marshal(gList)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	// TODO(tessr): Make this safe for concurrent updates. Will likely require a
	// conditional write operation for raftDB
	err = raftDB.Set(ctx, grantPrefix+grant.Policy, val)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &grant, nil
}

func EqualGrants(a, b Grant) bool {
	return a.GuardType == b.GuardType && bytes.Equal(a.GuardData, b.GuardData) && a.Protected == b.Protected
}
