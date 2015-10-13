package utxodb

import (
	"container/heap"
	"testing"
	"time"
)

func TestPoolReserveErr(t *testing.T) {
	p := &pool{
		ready: true,
	}
	u := &UTXO{Amount: 1}
	heap.Push(&p.outputs, u)

	now := time.Now()
	p.reserve(2, now, now.Add(time.Minute))
	if g := len(p.outputs); g != 1 {
		t.Errorf("len(p.outputs) = %d want 1", g)
	}
}

func TestReservedVsInsuf(t *testing.T) {
	t0 := time.Now()
	u := &UTXO{
		Amount:      10,
		ResvExpires: t0.Add(2 * time.Second),
		reserved:    5,
	}
	cases := []struct {
		amount uint64
		t      time.Time
		want   error
	}{
		{
			amount: 5,
			t:      t0,
			want:   ErrReserved,
		},
		{
			amount: 6,
			t:      t0,
			want:   ErrInsufficient,
		},
	}

	for _, test := range cases {
		u1 := new(UTXO)
		*u1 = *u
		p := &pool{
			outputs: []*UTXO{u1},
			ready:   true,
		}

		_, err := p.reserve(test.amount, test.t, test.t.Add(time.Minute))
		if err != test.want {
			t.Errorf("p.reserve(%d, %v) err = %v want %v", test.amount, test.t, err, test.want)
		}
	}
}
