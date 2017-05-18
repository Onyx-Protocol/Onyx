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

// Storage provides persistent storage for grant objects.
type Storage struct {
	sdb       *sinkdb.DB
	keyPrefix string
}

// NewStorage returns a new *Storage storing grants
// in db under keyPrefix.
// It implements the Loader interface.
func NewStorage(db *sinkdb.DB, keyPrefix string) *Storage {
	return &Storage{db, keyPrefix}
}

// Load satisfies the Loader interface.
func (s *Storage) Load(ctx context.Context, policy []string) ([]*Grant, error) {
	var grants []*Grant
	for _, p := range policy {
		var grantList GrantList
		found, err := s.sdb.GetStale(s.keyPrefix+p, &grantList)
		if err != nil {
			return nil, err
		} else if found {
			grants = append(grants, grantList.Grants...)
		}
	}
	return grants, nil
}

// Store stores g.
// If a grant equivalent to g is already stored,
// Store has no effect and returns a copy of the existing grant.
// Otherwise, if successful, it returns g.
func (s *Storage) Store(ctx context.Context, g *Grant) (*Grant, error) {
	key := s.keyPrefix + g.Policy
	if g.CreatedAt == "" {
		g.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	var grantList GrantList
	_, err := s.sdb.Get(ctx, key, &grantList)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	grants := grantList.Grants
	for _, existing := range grants {
		if EqualGrants(*existing, *g) {
			// this grant already exists, return for idempotency
			return existing, nil
		}
	}

	// create new grant and it append to the list of grants associated with this policy
	grants = append(grants, g)

	// TODO(tessr): Make this safe for concurrent updates. Will likely require a
	// conditional write operation for raftDB
	err = s.sdb.Exec(ctx, sinkdb.Set(s.keyPrefix+g.Policy, &GrantList{Grants: grants}))
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return g, nil
}

func EqualGrants(a, b Grant) bool {
	return a.GuardType == b.GuardType && bytes.Equal(a.GuardData, b.GuardData) && a.Protected == b.Protected
}
