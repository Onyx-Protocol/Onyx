// Package leader implements leader election between cored processes
// of a Chain Core.
package leader

import (
	"context"
	"sync/atomic"
	"time"

	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/log"
)

var isLeading atomic.Value

// IsLeading returns true if this process is
// the core leader.
func IsLeading() bool {
	v := isLeading.Load()
	if v == nil {
		return false
	}
	return v.(bool)
}

// Run runs as a goroutine, trying once every five seconds to become
// the leader for the core.  If it succeeds, then it calls the
// function lead (for generating or fetching blocks, and for
// expiring reservations) and enters a leadership-keepalive loop.
//
// Function lead is called when the local process becomes the leader.
// Its context is canceled when the process is deposed as leader.
//
// The Chain Core has up to a 10-second refractory period after
// shutdown, during which no process can become the new leader.
func Run(db *sql.DB, addr string, lead func(context.Context)) {
	ctx := context.Background()
	// We use our process's address as the key, because it's unique
	// among all processes within a Core and it allows a restarted
	// leader to immediately return to its leadership.
	l := &leader{
		db:      db,
		key:     addr,
		lead:    lead,
		address: addr,
	}
	log.Messagef(ctx, "Using leaderKey: %q", l.key)

	var leadCtx context.Context
	var cancel func()
	for leader := range leadershipChanges(ctx, l) {
		if leader {
			log.Messagef(ctx, "I am the core leader")
			leadCtx, cancel = context.WithCancel(ctx)
			l.lead(leadCtx)
		} else {
			log.Messagef(ctx, "No longer core leader")
			cancel()
		}

		isLeading.Store(leader)
	}
	panic("unreachable")
}

// leadershipChanges spawns a goroutine to check if this process
// is leader periodically. Every time the process becomes leader
// or is demoted from being a leader, it sends a bool on the
// returned channel.
func leadershipChanges(ctx context.Context, l *leader) chan bool {
	ch := make(chan bool)
	go func() {
		ticks := time.Tick(5 * time.Second)

		for {
			for !tryForLeadership(ctx, l) {
				<-ticks
			}
			ch <- true // elected leader

			for maintainLeadership(ctx, l) {
				<-ticks
			}
			ch <- false // demoted
		}
	}()
	return ch
}

type leader struct {
	// config
	db      *sql.DB
	key     string
	lead    func(context.Context)
	address string
}

func tryForLeadership(ctx context.Context, l *leader) bool {
	const insertQ = `
		INSERT INTO leader (leader_key, address, expiry) VALUES ($1, $2, CURRENT_TIMESTAMP + INTERVAL '10 seconds')
		ON CONFLICT (singleton) DO UPDATE SET leader_key = $1, address = $2, expiry = CURRENT_TIMESTAMP + INTERVAL '10 seconds'
			WHERE leader.expiry < CURRENT_TIMESTAMP
	`

	// Try to put this process's key into the leader table.  It
	// succeeds if the table's empty or the existing row (there can be
	// only one) is expired.  It fails otherwise.
	//
	// On success, this process's leadership expires in 10 seconds
	// unless it's renewed in the UPDATE query above.
	// That extends it for another 10 seconds.
	res, err := l.db.Exec(ctx, insertQ, l.key, l.address)
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

func maintainLeadership(ctx context.Context, l *leader) bool {
	const updateQ = `
		UPDATE leader SET expiry = CURRENT_TIMESTAMP + INTERVAL '10 seconds'
		WHERE leader_key = $1
	`

	res, err := l.db.Exec(ctx, updateQ, l.key)
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

// Address retrieves the IP address of the current
// core leader.
func Address(ctx context.Context, db pg.DB) (string, error) {
	const q = `SELECT address FROM leader`

	var addr string
	err := db.QueryRow(ctx, q).Scan(&addr)
	if err != nil {
		return "", errors.Wrap(err, "could not fetch leader address")
	}

	return addr, nil
}
