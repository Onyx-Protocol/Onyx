package errors

import (
	"io"
	"testing"
)

func TestReader(t *testing.T) {
	errX := New("x")
	tw := testReader{nil, errX, nil}
	r := Reader{R: &tw}
	_, err := r.Read([]byte{0})
	if err != nil {
		t.Error("unexpected error", err)
	}
	if g := r.N; g != 1 {
		t.Errorf("r.N = %d want 1", g)
	}
	if len(tw) != 2 {
		t.Errorf("len(tw) = %d want 2", len(tw))
	}
	for i := 0; i < 10; i++ {
		_, err = r.Read([]byte{0})
		if err != errX {
			t.Errorf("err = %v want %v", err, errX)
		}
		if g := r.N; g != 2 {
			t.Errorf("r.N = %d want 2", g)
		}
		if len(tw) != 1 {
			t.Errorf("len(tw) = %d want 1", len(tw))
		}
	}
	if got := r.Err; got != errX {
		t.Errorf("r.Err = %v want %v", got, errX)
	}
}

// testReader returns its errors in order.
// elements of a testReader may be nil.
// if its len is 0, it returns io.EOF.
type testReader []error

func (tw *testReader) Read(p []byte) (int, error) {
	if len(*tw) == 0 {
		return 0, io.EOF
	}
	err := (*tw)[0]
	*tw = (*tw)[1:]
	return len(p), err
}
