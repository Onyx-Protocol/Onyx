package authz

import (
	"bytes"
	"context"
	"time"

	"chain/database/sinkdb"
	"chain/errors"
)

// Generate code for the Grant and GrantList types.
//go:generate protoc -I. -I$CHAIN/.. --go_out=. grant.proto

// StoreGrant stores a new grant in sinkdb.
func StoreGrant(ctx context.Context, sdb *sinkdb.DB, grant Grant, grantPrefix string) (*Grant, error) {
	key := grantPrefix + grant.Policy
	if grant.CreatedAt == "" {
		grant.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	var grantList GrantList
	_, err := sdb.Get(ctx, key, &grantList)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	grants := grantList.GetGrants()
	for _, existing := range grants {
		if EqualGrants(*existing, grant) {
			// this grant already exists, return for idempotency
			return existing, nil
		}
	}

	// create new grant and it append to the list of grants associated with this policy
	grants = append(grants, &grant)

	// TODO(tessr): Make this safe for concurrent updates. Will likely require a
	// conditional write operation for raftDB
	err = sdb.Exec(ctx, sinkdb.Set(grantPrefix+grant.Policy, &GrantList{Grants: grants}))
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return &grant, nil
}

func EqualGrants(a, b Grant) bool {
	return a.GuardType == b.GuardType && bytes.Equal(a.GuardData, b.GuardData) && a.Protected == b.Protected
}
