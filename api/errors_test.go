package api

import (
	"chain/database/pg"
	"chain/errors"
	"database/sql"
	"testing"
)

func TestErrInfo(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 500},
		{sql.ErrNoRows, 500},
		{pg.ErrUserInputNotFound, 404},
		{errors.Wrap(pg.ErrUserInputNotFound, "foo"), 404},
		{sliceError{}, 500},
	}

	for _, test := range cases {
		_, info := errInfo(test.err)
		got := info.HTTPStatus
		if got != test.want {
			t.Errorf("errInfo(%#v) = %d want %d", test.err, got, test.want)
		}
	}
}

// Dummy error type, to test that errInfo
// doesn't panic when it's used as a map key.
type sliceError []int

func (err sliceError) Error() string { return "slice error" }
