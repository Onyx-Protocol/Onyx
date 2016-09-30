package core

import (
	"testing"

	"chain/core/query"
	"chain/errors"
)

func TestTxConsumerIsBefore(t *testing.T) {
	cases := []struct {
		a       string
		b       string
		wantRes bool
		wantErr error
	}{
		{"1:1-2", "1:2-3", true, nil},
		{"1:1-2", "2:2-3", true, nil},
		{"2:1-2", "1:2-3", false, nil},
		{"not-a-consumer", "also, not a consumer", false, query.ErrBadAfter},
	}

	for _, c := range cases {
		res, err := txAfterIsBefore(c.a, c.b)
		if errors.Root(err) != c.wantErr {
			t.Errorf("wanted err=%s, got %s", c.wantErr, err)
		}

		if res != c.wantRes {
			t.Errorf("wanted isBefore(%s, %s)=%t, got %t", c.a, c.b, c.wantRes, res)
		}
	}
}
