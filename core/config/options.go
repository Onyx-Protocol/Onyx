package config

import (
	"context"
	"path/filepath"

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
// options.
type Options struct {
	sdb    *sinkdb.DB
	schema map[string]option
}

// CleanFunc is implemented when defining a new configuration option.
// Implementations should canonicalize and validate newTuple. In-place
// modifications are ok.
type CleanFunc func(newTuple []string) error

// EqualFunc is implemented when defining a new set configuration option.
// Both a and b are guaranteed to be valid, canonicalized tuples.
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
	var set ValueSet
	_, err := opts.sdb.Get(ctx, filepath.Join(sinkdbPrefix, key), &set)
	if err != nil {
		return nil, err
	}
	var tuples [][]string
	for _, tup := range set.Tuples {
		tuples = append(tuples, tup.Values)
	}
	return tuples, nil
}

// Add adds the provided tuple to the configuration option set indicated
// by key.
func (opts *Options) Add(key string, tup []string) (sinkdb.Op, error) {
	opt, ok := opts.schema[key]
	if !ok {
		return sinkdb.Op{}, errors.WithDetailf(ErrConfigOp, "Configuration option %q is undefined.", key)
	}
	if opt.tupleSize != len(tup) {
		return sinkdb.Op{}, errors.WithDetailf(ErrConfigOp, "Configuration option %q expects %d arguments.", key, opt.tupleSize)
	}
	if opt.equalFunc == nil {
		return sinkdb.Op{}, errors.WithDetailf(ErrConfigOp, "Configuration option %q is a scalar. Use corectl set instead.")
	}

	// make a copy to avoid mutatating tup
	cleaned := make([]string, len(tup))
	copy(cleaned, tup)
	err := opt.cleanFunc(cleaned)
	if err != nil {
		return sinkdb.Op{}, errors.Sub(ErrConfigOp, err)
	}

	var existing ValueSet
	ver, err := opts.sdb.GetStale(filepath.Join(sinkdbPrefix, key), &existing)
	if err != nil {
		return sinkdb.Op{}, err
	}
	idx := findIndex(existing.Tuples, cleaned, opt.equalFunc)
	if idx != -1 {
		// tuple already exists, so the sinkdb op is a no-op
		return sinkdb.IfNotModified(ver), nil
	}

	// If the new tuple passed validation, then modify and write.
	modified := new(ValueSet)
	modified.Tuples = append(existing.Tuples, &ValueTuple{Values: cleaned})
	return sinkdb.All(
		sinkdb.IfNotModified(ver),
		sinkdb.Set(filepath.Join(sinkdbPrefix, key), modified),
	), nil
}

// Remove removes the provided tuple from the configuration option set
// indicated by key.
func (opts *Options) Remove(key string, tup []string) (sinkdb.Op, error) {
	opt, ok := opts.schema[key]
	if !ok {
		return sinkdb.Op{}, errors.WithDetailf(ErrConfigOp, "Configuration option %q undefined", key)
	}
	if opt.equalFunc == nil {
		return sinkdb.Op{}, errors.WithDetailf(ErrConfigOp, "Configuration option %q is a scalar. Use corectl set instead.")
	}

	// make a copy to avoid mutatating tup
	cleaned := make([]string, len(tup))
	copy(cleaned, tup)
	err := opt.cleanFunc(cleaned)
	if err != nil {
		return sinkdb.Op{}, errors.Sub(ErrConfigOp, err)
	}

	var existing ValueSet
	ver, err := opts.sdb.GetStale(filepath.Join(sinkdbPrefix, key), &existing)
	if err != nil {
		return sinkdb.Op{}, err
	}

	idx := findIndex(existing.Tuples, cleaned, opt.equalFunc)
	if idx == -1 {
		// tuple doesn't exists, so the sinkdb op is a no-op
		return sinkdb.IfNotModified(ver), nil
	}

	// Remove the tuple at the index from the set.
	modified := new(ValueSet)
	modified.Tuples = append(modified.Tuples, existing.Tuples[:idx]...)
	modified.Tuples = append(modified.Tuples, existing.Tuples[idx+1:]...)
	return sinkdb.All(
		sinkdb.IfNotModified(ver),
		sinkdb.Set(filepath.Join(sinkdbPrefix, key), modified),
	), nil
}

func findIndex(set []*ValueTuple, search []string, equal func([]string, []string) bool) int {
	for idx, tup := range set {
		if equal(tup.Values, search) {
			return idx
		}
	}
	return -1
}
