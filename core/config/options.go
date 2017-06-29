package config

import (
	"context"
	"fmt"
	"path"

	"chain/core/config/internal/configpb"
	"chain/database/sinkdb"
	"chain/errors"
)

const sinkdbPrefix = "/core/config/"

var ErrConfigOp = errors.New("invalid config operation")

func New(sdb *sinkdb.DB) *Options {
	return &Options{
		sdb:    sdb,
		schema: make(map[string]option),
	}
}

// Options provides access to Chain Core configuration options. All
// options should be defined before accessing and modifying any
// of their values.
type Options struct {
	sdb    *sinkdb.DB
	schema map[string]option
}

// CleanFunc is required when defining a new configuration option.
// Implementations should validate and canonicalize newTuple in-place.
type CleanFunc func(newTuple []string) error

// EqualFunc is required when defining a new set configuration option.
// Both a and b are guaranteed to have already been cleaned by the
// option's CleanFunc. It is also guaranteed that len(a) == len(b).
type EqualFunc func(a, b []string) bool

type option struct {
	tupleSize int
	cleanFunc CleanFunc
	equalFunc EqualFunc
}

// DefineSet defines a new configuration option as a set of tuples of
// size tupleSize. New tuples will be validated and canonicalized by
// cleanFunc. Equality when adding and removing is determined by equalFunc.
func (opts *Options) DefineSet(key string, tupleSize int, cleanFunc CleanFunc, equalFunc EqualFunc) {
	opts.schema[key] = option{
		tupleSize: tupleSize,
		cleanFunc: cleanFunc,
		equalFunc: equalFunc,
	}
}

// List returns all of the tuples for the provided configuration option.
func (opts *Options) List(ctx context.Context, key string) ([][]string, error) {
	if _, ok := opts.schema[key]; !ok {
		return nil, errors.WithDetailf(ErrConfigOp, "Configuration option %q is undefined.", key)
	}
	var set configpb.ValueSet
	_, err := opts.sdb.Get(ctx, path.Join(sinkdbPrefix, key), &set)
	if err != nil {
		return nil, err
	}
	var tuples [][]string
	for _, tup := range set.Tuples {
		tuples = append(tuples, tup.Values)
	}
	return tuples, nil
}

// ListFunc returns a closure that returns the set of tuples for the
// provided key.
//
// The configuration option for key must be a set of tuples.
// ListFunc will panic if the provided key is undefined in the schema or
// is defined as a scalar. The returned function will perform a stale
// read of the configuration value.
func (opts *Options) ListFunc(key string) func() ([][]string, error) {
	opt, ok := opts.schema[key]
	if !ok {
		panic(fmt.Errorf("unknown config option %q", key))
	} else if opt.equalFunc == nil {
		panic(fmt.Errorf("config option %q is a scalar, not a set", key))
	}

	return func() ([][]string, error) {
		var set configpb.ValueSet
		_, err := opts.sdb.GetStale(path.Join(sinkdbPrefix, key), &set)
		if err != nil {
			return nil, err
		}
		var tuples [][]string
		for _, tup := range set.Tuples {
			tuples = append(tuples, tup.Values)
		}
		return tuples, nil
	}
}

// Add adds the provided tuple to the configuration option set indicated
// by key.
func (opts *Options) Add(key string, tup []string) sinkdb.Op {
	opt, ok := opts.schema[key]
	if !ok {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q is undefined.", key))
	}
	if opt.tupleSize != len(tup) {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q expects %d arguments.", key, opt.tupleSize))
	}
	if opt.equalFunc == nil {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q is a scalar. Use corectl set instead."))
	}

	// make a copy to avoid mutating tup
	cleaned := make([]string, len(tup))
	copy(cleaned, tup)
	err := opt.cleanFunc(cleaned)
	if err != nil {
		return sinkdb.Error(errors.Sub(ErrConfigOp, err))
	}

	var existing configpb.ValueSet
	ver, err := opts.sdb.GetStale(path.Join(sinkdbPrefix, key), &existing)
	if err != nil {
		return sinkdb.Error(err)
	}
	idx := tupleIndex(existing.Tuples, cleaned, opt.equalFunc)
	if idx != -1 {
		// tuple already exists, so the sinkdb op is a no-op
		return sinkdb.IfNotModified(ver)
	}

	// If the new tuple passed validation, then modify and write.
	modified := new(configpb.ValueSet)
	modified.Tuples = append(existing.Tuples, &configpb.ValueTuple{Values: cleaned})
	return sinkdb.All(
		sinkdb.IfNotModified(ver),
		sinkdb.Set(path.Join(sinkdbPrefix, key), modified),
	)
}

// Remove removes the provided tuple from the configuration option set
// indicated by key.
func (opts *Options) Remove(key string, tup []string) sinkdb.Op {
	opt, ok := opts.schema[key]
	if !ok {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q undefined", key))
	}
	if opt.tupleSize != len(tup) {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q expects %d arguments.", key, opt.tupleSize))
	}
	if opt.equalFunc == nil {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q is a scalar. Use corectl set instead."))
	}

	// make a copy to avoid mutating tup
	cleaned := make([]string, len(tup))
	copy(cleaned, tup)
	err := opt.cleanFunc(cleaned)
	if err != nil {
		return sinkdb.Error(errors.Sub(ErrConfigOp, err))
	}

	var existing configpb.ValueSet
	ver, err := opts.sdb.GetStale(path.Join(sinkdbPrefix, key), &existing)
	if err != nil {
		return sinkdb.Error(err)
	}

	idx := tupleIndex(existing.Tuples, cleaned, opt.equalFunc)
	if idx == -1 {
		// tuple doesn't exist, so the sinkdb op is a no-op
		return sinkdb.IfNotModified(ver)
	}

	// Remove the tuple at the index from the set.
	modified := new(configpb.ValueSet)
	modified.Tuples = append(modified.Tuples, existing.Tuples[:idx]...)
	modified.Tuples = append(modified.Tuples, existing.Tuples[idx+1:]...)
	return sinkdb.All(
		sinkdb.IfNotModified(ver),
		sinkdb.Set(path.Join(sinkdbPrefix, key), modified),
	)
}

func tupleIndex(set []*configpb.ValueTuple, search []string, equal func(a, b []string) bool) int {
	for i, tup := range set {
		if equal(tup.Values, search) {
			return i
		}
	}
	return -1
}
