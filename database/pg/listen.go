package pg

import (
	"time"

	"golang.org/x/net/context"

	"github.com/lib/pq"

	"chain/errors"
	"chain/log"
)

// NewListener creates a new pq.Listener and begins listening.
func NewListener(ctx context.Context, dbURL, channel string) (*pq.Listener, error) {
	result := pq.NewListener(dbURL, 1*time.Second, 10*time.Second, func(ev pq.ListenerEventType, err error) {
		log.Error(ctx, errors.Wrapf(err, "event in %s listener: %v", channel, ev))
	})
	err := result.Listen(channel)
	return result, errors.Wrap(err, "listening to channel")
}
