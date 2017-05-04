// Package leader implements leader election between cored processes
// of a Chain Core.
package leader

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
)

// ProcessState is an enum describing the current state of the
// process. A recovering process has become leader but is still
// recovering the blockchain state. Some functionality is not
// available until the process enters the Leading state.
type ProcessState int

const (
	Following ProcessState = iota
	Recovering
	Leading
)

func (ps ProcessState) String() string {
	switch ps {
	case Following:
		return "following"
	case Recovering:
		return "recovering"
	case Leading:
		return "leading"
	default:
		panic(fmt.Errorf("unknown process state %d", ps))
	}
}

// ErrNoLeader is returned from Address when no process is
// currently leader.
var ErrNoLeader = errors.New("no leader process")

// Leader provides access to the Core leader process.
type Leader struct {
	state atomic.Value

	// config
	db      pg.DB
	key     string
	lead    func(context.Context)
	address string
}

// Address retrieves a routable address of the current
// Core leader.
func (l *Leader) Address(ctx context.Context) (string, error) {
	var addr string
	err := l.db.QueryRowContext(ctx, `SELECT address FROM leader`).Scan(&addr)
	if err == sql.ErrNoRows {
		return "", ErrNoLeader
	} else if err != nil {
		return "", errors.Wrap(err, "could not fetch leader address")
	}
	return addr, nil
}

// State returns the current state of this process.
func (l *Leader) State() ProcessState {
	v := l.state.Load()
	if v == nil {
		return Following
	}
	return v.(ProcessState)
}

// Run starts a goroutine, trying once every five seconds to become
// the leader for the core.  If it succeeds, then it calls the
// function lead (for generating or fetching blocks, and for
// expiring reservations) and enters a leadership-keepalive loop.
//
// Function lead is called when the local process becomes the leader.
// Its context is canceled when the process is deposed as leader.
//
// Run returns a pointer to a Leader struct that can be queried to
// check the state of the process or find the current leader.
//
// The Chain Core has up to a 1.5-second refractory period after
// shutdown, during which no process may be leader.
func Run(ctx context.Context, db pg.DB, addr string, lead func(context.Context)) *Leader {
	// We use our process's address as the key, because it's unique
	// among all processes within a Core and it allows a restarted
	// leader to immediately return to its leadership.
	l := &Leader{
		db:      db,
		key:     addr,
		lead:    lead,
		address: addr,
	}
	log.Printf(ctx, "Using leaderKey: %q", l.key)

	go func() {
		cancel := func() {}
		var leadCtx context.Context
		for leader := range leadershipChanges(ctx, l) {
			if leader {
				log.Printf(ctx, "I am the core leader")
				l.state.Store(Recovering)
				leadCtx, cancel = context.WithCancel(ctx)
				l.lead(leadCtx)
				l.state.Store(Leading)
			} else {
				log.Printf(ctx, "No longer core leader")
				l.state.Store(Following)
				cancel()
			}
		}
		cancel()
	}()
	return l
}

// leadershipChanges spawns a goroutine to check if this process
// is leader periodically. Every time the process becomes leader
// or is demoted from being a leader, it sends a bool on the
// returned channel.
//
// It provides the invariants:
// * The first value sent on the channel is true. (This will
//   happen at the time the process is first elected leader.)
// * Every value sent on the channel is the opposite of the
//   previous value.
func leadershipChanges(ctx context.Context, l *Leader) chan bool {
	ch := make(chan bool)
	go func() {
		ticks := time.Tick(500 * time.Millisecond)

		for {
			for !tryForLeadership(ctx, l) {
				// Wait for a tick of the ticker or the context
				// to be cancelled.
				select {
				case <-ctx.Done():
					close(ch)
					return
				case <-ticks:
				}
			}
			ch <- true // elected leader

			for maintainLeadership(ctx, l) {
				// Wait for a tick of the ticker or the context
				// to be cancelled.
				select {
				case <-ctx.Done():
					close(ch)
					return
				case <-ticks:
				}
			}
			ch <- false // demoted
		}
	}()
	return ch
}

func tryForLeadership(ctx context.Context, l *Leader) bool {
	const insertQ = `
		INSERT INTO leader (leader_key, address, expiry) VALUES ($1, $2, CURRENT_TIMESTAMP + INTERVAL '1 second')
		ON CONFLICT (singleton) DO UPDATE SET leader_key = $1, address = $2, expiry = CURRENT_TIMESTAMP + INTERVAL '1 second'
			WHERE leader.expiry < CURRENT_TIMESTAMP
	`

	// Try to put this process's key into the leader table.  It
	// succeeds if the table's empty or the existing row (there can be
	// only one) is expired.  It fails otherwise.
	//
	// On success, this process's leadership expires in 1 second
	// unless it's renewed in the UPDATE query in maintainLeadership.
	// That extends it for another 1 second.
	res, err := l.db.ExecContext(ctx, insertQ, l.key, l.address)
	if err != nil {
		log.Error(ctx, err)
		return false
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error(ctx, err)
		return false
	}
	return rowsAffected > 0
}

func maintainLeadership(ctx context.Context, l *Leader) bool {
	const updateQ = `
		UPDATE leader SET expiry = CURRENT_TIMESTAMP + INTERVAL '1 second'
		WHERE leader_key = $1
	`

	res, err := l.db.ExecContext(ctx, updateQ, l.key)
	if err != nil {
		log.Error(ctx, err)
		return false
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error(ctx, err)
		return false
	}
	return rowsAffected > 0
}
