package bc

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestBlockHeaderValid(t *testing.T) {
	base := NewBlockHeaderEntry(1, 1, Hash{}, 1, Hash{}, Hash{}, nil)

	var bh BlockHeaderEntry

	cases := []struct {
		f   func()
		err error
	}{
		{},
		{
			f: func() {
				bh.Body.Version = 2
			},
		},
		{
			f: func() {
				bh.Body.ExtHash = Hash{1}
			},
			err: errNonemptyExtHash,
		},
	}

	for i, c := range cases {
		t.Logf("case %d", i)
		bh = *base
		if c.f != nil {
			c.f()
		}
		err := bh.CheckValid(nil)
		if err != c.err {
			t.Errorf("case %d: got error %s, want %s; bh is:\n%s", i, err, c.err, spew.Sdump(bh))
		}
	}
}
