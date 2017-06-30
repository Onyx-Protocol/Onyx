package config

import (
	"context"
	"reflect"
	"testing"

	"chain/database/sinkdb/sinkdbtest"
)

func identityFunc(tup []string) error    { return nil }
func reflectEquality(a, b []string) bool { return reflect.DeepEqual(a, b) }

func must(t testing.TB, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetTuples(t *testing.T) {
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

func TestListFunc(t *testing.T) {
	sdb := sinkdbtest.NewDB(t)
	opts := New(sdb)
	opts.DefineSet("example", 1, identityFunc, reflectEquality)

	ctx := context.Background()

	must(t, sdb.Exec(ctx, opts.Add("example", []string{"foo"})))
	must(t, sdb.Exec(ctx, opts.Add("example", []string{"bar"})))

	// perform a linearizable read since ListFunc won't and we
	// want a deterministic test case
	_, err := opts.List(ctx, "example")
	must(t, err)

	got := opts.ListFunc("example")()
	want := [][]string{{"foo"}, {"bar"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}
