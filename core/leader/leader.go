// Package leader implements leader election between cored processes
// of a Chain Core.
package leader

import (
	"context"
	"sync"
	"time"

	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/log"
)

type Leadership interface {
	IsLeading() bool
	Address(ctx context.Context) (string, error)
}

type leadership struct {
	db      pg.DB
	key     string
	lead    func(context.Context)
	address string

	// state
	mu      sync.Mutex
	leading bool
	cancel  func()
}

// IsLeading returns true if this process is the core leader.
func (l *leadership) IsLeading() bool {
	l.mu.Lock()
	leading := l.leading
	l.mu.Unlock()
	return leading
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
func Run(db *sql.DB, addr string, lead func(context.Context)) Leadership {
	ctx := context.Background()
	// We use our process's address as the key, because it's unique
	// among all processes within a Core and it allows a restarted
	// leader to immediately return to its leadership.
	l := &leadership{
		db:      db,
		key:     addr,
		lead:    lead,
		address: addr,
	}
	log.Messagef(ctx, "Using leaderKey: %q", l.key)

	l.update(ctx)
	go func() {
		for range time.Tick(5 * time.Second) {
			l.update(ctx)
		}
	}()
	return l
}

func (l *leadership) update(ctx context.Context) {
	const (
		insertQ = `
			INSERT INTO leader (leader_key, address, expiry) VALUES ($1, $2, CURRENT_TIMESTAMP + INTERVAL '10 seconds')
			ON CONFLICT (singleton) DO UPDATE SET leader_key = $1, address = $2, expiry = CURRENT_TIMESTAMP + INTERVAL '10 seconds'
				WHERE leader.expiry < CURRENT_TIMESTAMP
		`
		updateQ = `
			UPDATE leader SET expiry = CURRENT_TIMESTAMP + INTERVAL '10 seconds'
				WHERE leader_key = $1
		`
	)

	if l.IsLeading() {
		res, err := l.db.Exec(ctx, updateQ, l.key)
		if err == nil {
			rowsAffected, err := res.RowsAffected()
			if err == nil && rowsAffected > 0 {
				// still leading
				return
			}
		}

		// Either the UPDATE affected no rows, or it (or RowsAffected)
		// produced an error.

		if err != nil {
			log.Error(ctx, err)
		}
		log.Messagef(ctx, "No longer core leader")
		l.mu.Lock()
		l.cancel()
		l.leading = false
		l.cancel = nil
		l.mu.Unlock()
	} else {
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
			return
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			log.Error(ctx, err)
			return
		}

		if rowsAffected == 0 {
			return
		}

		log.Messagef(ctx, "I am the core leader")

		ctx, cancel := context.WithCancel(ctx)
		go l.lead(ctx)

		l.mu.Lock()
		l.leading = true
		l.cancel = cancel
		l.mu.Unlock()
	}
}

// Address retrieves the IP address of the current
// core leader.
func (l *leadership) Address(ctx context.Context) (string, error) {
	const q = `SELECT address FROM leader`

	var addr string
	err := l.db.QueryRow(ctx, q).Scan(&addr)
	if err != nil {
		return "", errors.Wrap(err, "could not fetch leader address")
	}

	return addr, nil
}
