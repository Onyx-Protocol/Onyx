package errors

import (
	"errors"
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
