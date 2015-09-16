package errors

import (
	"io"
	"testing"
)

func TestWriter(t *testing.T) {
	errX := New("x")
	tw := testWriter{nil, errX, nil}
	w := NewWriter(&tw)
	_, err := w.Write([]byte{1})
	if err != nil {
		t.Error("unexpected error", err)
	}
	if g := w.Written(); g != 1 {
		t.Errorf("w.Written() = %d want 1", g)
	}
	if len(tw) != 2 {
		t.Errorf("len(tw) = %d want 2", len(tw))
	}
	for i := 0; i < 10; i++ {
		_, err = w.Write([]byte{1})
		if err != errX {
			t.Errorf("err = %v want %v", err, errX)
		}
		if g := w.Written(); g != 2 {
			t.Errorf("w.Written() = %d want 2", g)
		}
		if len(tw) != 1 {
			t.Errorf("len(tw) = %d want 1", len(tw))
		}
	}
	if got := w.Err(); got != errX {
		t.Errorf("w.Err() = %v want %v", got, errX)
	}
}

// testWriter returns its errors in order.
// elements of a testWriter may be nil.
// if its len is 0, it returns io.EOF.
type testWriter []error

func (tw *testWriter) Write(p []byte) (int, error) {
	if len(*tw) == 0 {
		return len(p), io.EOF
	}
	err := (*tw)[0]
	*tw = (*tw)[1:]
	return len(p), err
}
