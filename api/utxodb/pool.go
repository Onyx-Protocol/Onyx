package utxodb

import (
	"container/heap"
	"sort"
	"sync"
	"time"

	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/metrics"
)

// A pool holds outputs of a single asset type in an account.
type pool struct {
	mu      sync.Mutex // protects the following
	ready   bool
	outputs utxosByResvExpires // min heap
}

func (p *pool) init(ctx context.Context, db DB, k key) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ready {
		return nil
	}

	defer metrics.RecordElapsed(time.Now())

	utxos, err := db.LoadUTXOs(ctx, k.AccountID, k.AssetID)
	if err != nil {
		return err
	}

	internIDs(utxos)
	utxos = append(utxos, p.outputs...)
	p.outputs = nil
	sort.Sort(byOutpoint(utxos))

	for i, utxo := range utxos {
		if i > 0 && utxos[i-1].Outpoint == utxo.Outpoint {
			continue
		}
		heap.Push(&p.outputs, utxo)
	}
	p.ready = true
	return nil
}

// reserve reserves UTXOs from p to satisfy in and returns them.
// If the input can't be satisfied, it returns nil.
func (p *pool) reserve(amount uint64, now, exp time.Time) ([]*UTXO, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	defer metrics.RecordElapsed(time.Now())

	// Put all collected utxos back in the heap,
	// no matter what. They may or may not have
	// ResvExpires changed.
	var utxos []*UTXO
	defer func() {
		for _, utxo := range utxos {
			heap.Push(&p.outputs, utxo)
		}
	}()

	// TODO(kr): handle reserve-by-txid

	var avail, change uint64
	for len(p.outputs) > 0 {
		u := heap.Pop(&p.outputs).(*UTXO)
		utxos = append(utxos, u)
		if u.ResvExpires.After(now) {
			// We cannot satisfy the request now, but we should
			// still check if there's enough money in the account,
			// counting reserved outputs. This lets us discriminate
			// between "you don't have enough money" (ErrInsufficient)
			// vs "you have enough money, but some of it is
			// locked up in a reservation and you have to wait
			// for a new change output before you can spend it"
			// (ErrReserved).
			change += u.Amount - u.reserved
		} else {
			avail += u.Amount
		}
		if avail >= amount {
			// Success. Mark the collected utxos
			// with a reservation expiration time.
			for _, utxo := range utxos {
				utxo.ResvExpires = exp
				if amount < u.Amount {
					utxo.reserved = amount
				} else {
					utxo.reserved = u.Amount
				}
				amount -= utxo.reserved
			}
			return utxos, nil
		}
		if avail+change >= amount {
			return nil, ErrReserved
		}
	}
	return nil, ErrInsufficient
}

// caller must hold p.mu
func (p *pool) contains(u *UTXO) bool {
	i := u.heapIndex
	return i < len(p.outputs) && p.outputs[i] == u
}

// findReservation finds the UTXO in p that reserves op.
// If there is no such reservation, it returns nil.
func (p *pool) findReservation(op bc.Outpoint) *UTXO {
	p.mu.Lock()
	defer p.mu.Unlock()
	defer metrics.RecordElapsed(time.Now())

	u := p.byOutpoint(op)
	if u == nil || time.Now().After(u.ResvExpires) {
		return nil
	}
	return u
}

func (p *pool) byOutpoint(op bc.Outpoint) *UTXO {
	for _, u := range p.outputs {
		if u.Outpoint == op {
			return u
		}
	}
	return nil
}
