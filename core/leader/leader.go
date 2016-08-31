package leader

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"chain/database/sql"
	"chain/log"
)

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
	leaderKeyBytes := make([]byte, 32)
	_, err := rand.Read(leaderKeyBytes)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}
	l := &leader{
		db:      db,
		key:     hex.EncodeToString(leaderKeyBytes),
		lead:    lead,
		address: addr,
	}
	log.Messagef(ctx, "Chose leaderKey: %s", l.key)

	update(ctx, l)
	for range time.Tick(5 * time.Second) {
		update(ctx, l)
	}
}

type leader struct {
	// config
	db      *sql.DB
	key     string
	lead    func(context.Context)
	address string

	// state
	leading bool
	cancel  func()
}

func update(ctx context.Context, l *leader) {
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

	if l.leading {
		res, err := l.db.Exec(ctx, updateQ, l.key, l.address)
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
		l.cancel()
		l.leading = false
		l.cancel = nil
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

		l.leading = true
		ctx, l.cancel = context.WithCancel(ctx)
		go l.lead(ctx)
	}
}
