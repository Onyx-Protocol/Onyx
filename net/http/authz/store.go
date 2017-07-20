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

// Store provides persistent storage for grant objects.
type Store struct {
	sdb       *sinkdb.DB
	keyPrefix string
}

// NewStore returns a new *Store storing grants
// in db under keyPrefix.
// It implements the Loader interface.
func NewStore(db *sinkdb.DB, keyPrefix string) *Store {
	return &Store{db, keyPrefix}
}

// Load satisfies the Loader interface.
func (s *Store) Load(ctx context.Context, policy []string) ([]*Grant, error) {
	var grants []*Grant
	for _, p := range policy {
		var grantList GrantList
		ver, err := s.sdb.GetStale(s.keyPrefix+p, &grantList)
		if err != nil {
			return nil, err
		} else if ver.Exists() {
			grants = append(grants, grantList.Grants...)
		}
	}
	return grants, nil
}

// Save returns an Op to store all the grants passed in as g,
// which must all have the same policy
// Duplicates are ignored
// It also sets field CreatedAt to the time the grants
// are stored, or the time the original grant was stored
func (s *Store) Save(ctx context.Context, g ...*Grant) sinkdb.Op {
	if len(g) == 0 {
		return sinkdb.Op{}
	}
	policy := g[0].Policy
	key := s.keyPrefix + policy
	var existing GrantList
	ver, err := s.sdb.Get(ctx, key, &existing)
	if err != nil {
		return sinkdb.Error(errors.Wrap(err))
	}
	var newGrants []*Grant
	for _, grant := range g {
		if grant.Policy != policy {
			return sinkdb.Error(errors.New("Grants have mismatching policies"))
		}
		if grant.CreatedAt == "" {
			grant.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		var include = true
		for _, e := range existing.Grants {
			if EqualGrants(*e, *grant) {
				include = false
			}
		}
		if include {
			newGrants = append(newGrants, grant)
		}
	}
	newGrants = append(existing.Grants, newGrants...)

	return sinkdb.All(
		sinkdb.IfNotModified(ver),
		sinkdb.Set(s.keyPrefix+policy, &GrantList{Grants: newGrants}),
	)
}

// Delete returns an Op to delete from policy all stored grants for which delete returns true.
func (s *Store) Delete(policy string, delete func(*Grant) bool) sinkdb.Op {
	key := s.keyPrefix + policy

	var grantList GrantList
	ver, err := s.sdb.GetStale(key, &grantList)
	if err != nil || !ver.Exists() {
		return sinkdb.Error(errors.Wrap(err)) // if !exists, errors.Wrap(err) is nil
	}

	var keep []*Grant
	for _, g := range grantList.Grants {
		if !delete(g) {
			keep = append(keep, g)
		}
	}

	// We didn't match any grants, don't need to do an update. Return no-op.
	if len(keep) == len(grantList.Grants) {
		return sinkdb.Op{}
	}

	return sinkdb.All(
		sinkdb.IfNotModified(ver),
		sinkdb.Set(key, &GrantList{Grants: keep}),
	)
}

func EqualGrants(a, b Grant) bool {
	return a.GuardType == b.GuardType && bytes.Equal(a.GuardData, b.GuardData) && a.Protected == b.Protected
}
