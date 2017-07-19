package config

import (
	"context"
	"reflect"
	"testing"

	"chain/database/sinkdb/sinkdbtest"
)

func identityFunc(tup []string) error    { return nil }
func reflectEquality(a, b []string) bool { return reflect.DeepEqual(a, b) }
func firstEqual(a, b []string) bool      { return a[0] == b[0] }

func must(t testing.TB, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestAdd(t *testing.T) {
	sdb := sinkdbtest.NewDB(t)
	opts := New(sdb)
	opts.DefineSet("example", 2, identityFunc, reflectEquality)

	ctx := context.Background()

	must(t, sdb.Exec(ctx, opts.Add("example", []string{"foo", "bar"})))
	must(t, sdb.Exec(ctx, opts.Add("example", []string{"baz", "bax"})))

	// duplicate write should be no-op
	must(t, sdb.Exec(ctx, opts.Add("example", []string{"foo", "bar"})))

	got, err := opts.List(ctx, "example")
	must(t, err)
	want := [][]string{
		{"foo", "bar"},
		{"baz", "bax"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}

	must(t, sdb.Exec(ctx, opts.Remove("example", []string{"foo", "bar"})))

	got, err = opts.List(ctx, "example")
	must(t, err)
	want = [][]string{{"baz", "bax"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestAddOrUpdate(t *testing.T) {
	sdb := sinkdbtest.NewDB(t)
	opts := New(sdb)
	opts.DefineSet("example", 2, identityFunc, firstEqual)

	ctx := context.Background()
	must(t, sdb.Exec(ctx, opts.AddOrUpdate("example", []string{"foo", "bar"})))
	must(t, sdb.Exec(ctx, opts.AddOrUpdate("example", []string{"foo", "baz"})))

	// Because equality is defined on the first value, "example" should
	// now have a single tuple in its set: (foo, baz).

	got, err := opts.List(ctx, "example")
	must(t, err)
	want := [][]string{{"foo", "baz"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestListFunc(t *testing.T) {
	sdb := sinkdbtest.NewDB(t)
	opts := New(sdb)
	opts.DefineSet("example", 1, identityFunc, reflectEquality)

	ctx := context.Background()

	must(t, sdb.Exec(ctx, opts.Add("example", []string{"foo"})))
	must(t, sdb.Exec(ctx, opts.Add("example", []string{"bar"})))

	// perform a linearizable read since ListFunc won't and we
	// want a deterministic test case
	must(t, sdb.RaftService().WaitRead(ctx))

	got := opts.ListFunc("example")()
	want := [][]string{{"foo"}, {"bar"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestSet(t *testing.T) {
	sdb := sinkdbtest.NewDB(t)
	opts := New(sdb)
	opts.DefineSingle("example", 2, identityFunc)

	ctx := context.Background()

	must(t, sdb.Exec(ctx, opts.Set("example", []string{"foo", "bar"})))
	got, err := opts.List(ctx, "example")
	must(t, err)
	want := [][]string{{"foo", "bar"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}

	must(t, sdb.Exec(ctx, opts.Set("example", []string{"baz", "bax"})))
	got, err = opts.List(ctx, "example")
	must(t, err)
	want = [][]string{{"baz", "bax"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}

	must(t, sdb.Exec(ctx, opts.Remove("example", []string{"baz", "bax"})))
	got, err = opts.List(ctx, "example")
	must(t, err)
	if got != nil {
		t.Errorf("got %#v, want nil", got)
	}
}

func TestGetFunc(t *testing.T) {
	sdb := sinkdbtest.NewDB(t)
	opts := New(sdb)
	opts.DefineSingle("example", 1, identityFunc)

	ctx := context.Background()
	must(t, sdb.Exec(ctx, opts.Set("example", []string{"foo"})))

	// perform a linearizable read since GetFunc won't and we
	// want a deterministic test case
	must(t, sdb.RaftService().WaitRead(ctx))

	got := opts.GetFunc("example")()
	want := []string{"foo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetFunc(\"example\")() = %#v, want %#v", got, want)
	}
}
