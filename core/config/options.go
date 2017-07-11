package config

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"sync/atomic"

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

	errsMu sync.Mutex
	errs   map[string]error
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

// Err returns any persistent errors encountered by ListFunc's closures
// when retrieving configuration values. If there are multiple errors,
// it'll return an arbitrary one.
func (opts *Options) Err() error {
	opts.errsMu.Lock()
	defer opts.errsMu.Unlock()

	// TODO(jackson): return all of the errors as a composite error?
	for key, err := range opts.errs {
		return fmt.Errorf("%q: %s", key, err.Error())
	}
	return nil
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

// DefineSingle defines a new configuration option that takes a single
// tuple value of size tupleSize. New tuples will be validated and
// canonicalized by cleanFunc.
func (opts *Options) DefineSingle(key string, tupleSize int, cleanFunc CleanFunc) {
	opts.schema[key] = option{
		tupleSize: tupleSize,
		cleanFunc: cleanFunc,
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

// ListFunc returns a closure that returns the set of tuples
// for the provided key.
//
// The configuration option for key must be a set of tuples.
// ListFunc panics if the provided key is undefined or is
// defined as a scalar.
//
// The returned function performs a stale read of the configuration
// value. If an error occurs while reading the value the old
// value is returned, and the error is saved on the Options
// type to be returned in Err.
func (opts *Options) ListFunc(key string) func() [][]string {
	opt, ok := opts.schema[key]
	if !ok {
		panic(fmt.Errorf("unknown config option %q", key))
	} else if opt.equalFunc == nil {
		panic(fmt.Errorf("config option %q is a scalar, not a set", key))
	}

	// old is the last successfully retrieved tuple set for this
	// configuration key. The returned closure updates it on
	// every successful lookup. If an error occurs, the closure
	// returns this value.
	var old atomic.Value
	old.Store([][]string(nil))

	return func() [][]string {
		var set configpb.ValueSet
		_, err := opts.sdb.GetStale(path.Join(sinkdbPrefix, key), &set)
		if err != nil {
			opts.errsMu.Lock()
			opts.errs[key] = err
			opts.errsMu.Unlock()
			return old.Load().([][]string)
		}

		// clear any error for this key bc we succeeded
		opts.errsMu.Lock()
		delete(opts.errs, key)
		opts.errsMu.Unlock()

		var tuples [][]string
		for _, tup := range set.Tuples {
			tuples = append(tuples, tup.Values)
		}
		old.Store(tuples)
		return tuples
	}
}

// GetFunc returns a closure that returns the tuple value
// for the provided key.
//
// The configuration option for key must be defined as a
// single tuple. GetFunc panics if the provided key is
// undefined or is defined as a set of tuples.
//
// The returned function performs a stale read of the configuration
// value. If an error occurs while reading the value the old
// value is returned, and the error is saved on the Options
// type to be returned in Err.
func (opts *Options) GetFunc(key string) func() []string {
	opt, ok := opts.schema[key]
	if !ok {
		panic(fmt.Errorf("unknown config option %q", key))
	} else if opt.equalFunc != nil {
		panic(fmt.Errorf("config option %q is a set, not a scalar", key))
	}

	// old is the last successfully retrieved tuple for this
	// configuration key. The returned closure updates it on
	// every successful lookup. If an error occurs, the closure
	// returns this value.
	var old atomic.Value
	old.Store([]string(nil))

	return func() []string {
		var set configpb.ValueSet
		_, err := opts.sdb.GetStale(path.Join(sinkdbPrefix, key), &set)
		if err != nil {
			opts.errsMu.Lock()
			opts.errs[key] = err
			opts.errsMu.Unlock()
			return old.Load().([]string)
		}

		// clear any error for this key bc we succeeded
		opts.errsMu.Lock()
		delete(opts.errs, key)
		opts.errsMu.Unlock()

		if len(set.Tuples) == 0 {
			old.Store([]string(nil))
			return nil
		}
		old.Store(set.Tuples[0].Values)
		return set.Tuples[0].Values
	}
}

// Add adds the provided tuple to the configuration option set indicated
// by key. If the added tuple conflicts with an existing tuple in the set,
// Add returns an error describing the conflict.
func (opts *Options) Add(key string, tup []string) sinkdb.Op {
	return opts.add(key, tup, func(new, existing []string) error {
		return errors.WithDetailf(ErrConfigOp,
			"Value (%s) conflicts with the existing value (%s)",
			strings.Join(new, " "),
			strings.Join(existing, " "))
	})
}

// AddOrUpdate adds the provided tuple to the configuration option
// set indicated by key. If the added tuple conflicts with an
// existing tuple in the set, AddOrUpdate updates the conflicting
// tuple to the provided tuple.
func (opts *Options) AddOrUpdate(key string, tup []string) sinkdb.Op {
	return opts.add(key, tup, func(new, existing []string) error {
		// overwrite the existing tuple with the new tuple
		for k, v := range new {
			existing[k] = v
		}
		return nil
	})
}

func (opts *Options) add(key string, tup []string, onConflict func(new, existing []string) error) sinkdb.Op {
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

	var set configpb.ValueSet
	ver, err := opts.sdb.GetStale(path.Join(sinkdbPrefix, key), &set)
	if err != nil {
		return sinkdb.Error(err)
	}

	idx := tupleIndex(set.Tuples, cleaned, opt.equalFunc)
	if idx == -1 {
		// tup is a new tuple so append it to the existing set
		set.Tuples = append(set.Tuples, &configpb.ValueTuple{Values: cleaned})
	} else if exactlyEqual(set.Tuples[idx].Values, cleaned) {
		// tup exactly matches an existing tuple, so this
		// is a no-op
		return sinkdb.IfNotModified(ver)
	} else {
		// tup's key matches an existing tuple but the rest of
		// tup does not match. use onConflict to determine how
		// to handle the conflict
		err = onConflict(cleaned, set.Tuples[idx].Values)
		if err != nil {
			return sinkdb.Error(err)
		}
	}
	return sinkdb.All(
		sinkdb.IfNotModified(ver),
		sinkdb.Set(path.Join(sinkdbPrefix, key), &set),
	)
}

// Set updates the configuration option indicated by key with the value
// tup. If the configuration option is already set, Set will overwrite
// the existing value.
func (opts *Options) Set(key string, tup []string) sinkdb.Op {
	opt, ok := opts.schema[key]
	if !ok {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q is undefined.", key))
	}
	if opt.tupleSize != len(tup) {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q expects %d arguments.", key, opt.tupleSize))
	}
	if opt.equalFunc != nil {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q is a set of tuples. Use corectl add instead."))
	}

	// make a copy to avoid mutating tup
	cleaned := make([]string, len(tup))
	copy(cleaned, tup)
	err := opt.cleanFunc(cleaned)
	if err != nil {
		return sinkdb.Error(errors.Sub(ErrConfigOp, err))
	}
	return sinkdb.Set(path.Join(sinkdbPrefix, key), &configpb.ValueSet{
		Tuples: []*configpb.ValueTuple{{Values: cleaned}},
	})
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
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q is a scalar. Use corectl unset instead."))
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

// Unset clears any value from the single-value configuration option
// indicated by key.
func (opts *Options) Unset(key string) sinkdb.Op {
	opt, ok := opts.schema[key]
	if !ok {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q undefined", key))
	}
	if opt.equalFunc != nil {
		return sinkdb.Error(errors.WithDetailf(ErrConfigOp, "Configuration option %q is a set. Use corectl rm instead."))
	}
	return sinkdb.Delete(path.Join(sinkdbPrefix, key))
}

func tupleIndex(set []*configpb.ValueTuple, search []string, equal func(a, b []string) bool) int {
	for i, tup := range set {
		if equal(tup.Values, search) {
			return i
		}
	}
	return -1
}

func exactlyEqual(a, b []string) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
