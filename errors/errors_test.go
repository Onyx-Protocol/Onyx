package errors

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestWrap(t *testing.T) {
	err := errors.New("0")
	err1 := Wrap(err, "1")
	err2 := Wrap(err1, "2")
	err3 := Wrap(err2)

	if got := Root(err1); got != err {
		t.Fatalf("Root(%v)=%v want %v", err1, got, err)
	}

	if got := Root(err2); got != err {
		t.Fatalf("Root(%v)=%v want %v", err2, got, err)
	}

	if err2.Error() != "2: 1: 0" {
		t.Fatalf("err msg = %s want '2: 1: 0'", err2.Error())
	}

	if err3.Error() != "2: 1: 0" {
		t.Fatalf("err msg = %s want '2: 1: 0'", err3.Error())
	}

	stack := Stack(err1)
	if len(stack) == 0 {
		t.Fatalf("len(stack) = %v want > 0", len(stack))
	}
	if !strings.Contains(stack[0].String(), "TestWrap") {
		t.Fatalf("first stack frame should contain \"TestWrap\": %v", stack[0].String())
	}

	if !reflect.DeepEqual(Stack(err2), Stack(err1)) {
		t.Errorf("err2 stack got %v want %v", Stack(err2), Stack(err1))
	}

	if !reflect.DeepEqual(Stack(err3), Stack(err1)) {
		t.Errorf("err3 stack got %v want %v", Stack(err3), Stack(err1))
	}
}

func TestWrapNil(t *testing.T) {
	var err error

	err1 := Wrap(err, "1")
	if err1 != nil {
		t.Fatal("wrapping nil error should yield nil")
	}
}

func TestWrapf(t *testing.T) {
	err := errors.New("0")
	err1 := Wrapf(err, "there are %d errors being wrapped", 1)
	if err1.Error() != "there are 1 errors being wrapped: 0" {
		t.Fatalf("err msg = %s want 'there are 1 errors being wrapped: 0'", err1.Error())
	}
}

func TestWrapMsg(t *testing.T) {
	err := errors.New("rooti")
	err1 := Wrap(err, "cherry", " ", "guava")
	if err1.Error() != "cherry guava: rooti" {
		t.Fatalf("err msg = %s want 'cherry guava: rooti'", err1.Error())
	}
}

func TestDetail(t *testing.T) {
	root := errors.New("foo")
	cases := []struct {
		err     error
		detail  string
		message string
	}{
		{root, "", "foo"},
		{WithDetail(root, "bar"), "bar", "bar: foo"},
		{WithDetail(WithDetail(root, "bar"), "baz"), "bar; baz", "baz: bar: foo"},
		{Wrap(WithDetail(root, "bar"), "baz"), "bar", "baz: bar: foo"},
	}

	for _, test := range cases {
		if got := Detail(test.err); got != test.detail {
			t.Errorf("Detail(%v) = %v want %v", test.err, got, test.detail)
		}
		if got := Root(test.err); got != root {
			t.Errorf("Root(%v) = %v want %v", test.err, got, root)
		}
		if got := test.err.Error(); got != test.message {
			t.Errorf("(%v).Error() = %v want %v", test.err, got, test.message)
		}
	}
}

func TestData(t *testing.T) {
	root := errors.New("foo")
	cases := []struct {
		err  error
		data interface{}
	}{
		{WithData(root, "a", "b"), map[string]interface{}{"a": "b"}},
		{WithData(WithData(root, "a", "b"), "c", "d"), map[string]interface{}{"a": "b", "c": "d"}},
		{Wrap(WithData(root, "a", "b"), "baz"), map[string]interface{}{"a": "b"}},
	}

	for _, test := range cases {
		if got := Data(test.err); !reflect.DeepEqual(got, test.data) {
			t.Errorf("Data(%#v) = %v want %v", test.err, got, test.data)
		}
		if got := Root(test.err); got != root {
			t.Errorf("Root(%#v) = %v want %v", test.err, got, root)
		}
	}
}
