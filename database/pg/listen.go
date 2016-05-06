package pg

import (
	"time"

	"golang.org/x/net/context"

	"github.com/lib/pq"

	"chain/errors"
	"chain/log"
	"chain/net"
)

// NewListener creates a new pq.Listener and begins listening.
func NewListener(ctx context.Context, dbURL, channel string) (*pq.Listener, error) {
	dbURL, err := resolveURI(dbURL)
	if err != nil {
		return nil, err
	}

	// We want etcd name lookups so we use our own Dialer.
	d := new(net.Dialer)
	result := pq.NewDialListener(d, dbURL, 1*time.Second, 10*time.Second, func(ev pq.ListenerEventType, err error) {
		log.Error(ctx, errors.Wrapf(err, "event in %s listener: %v", channel, ev))
	})
	err = result.Listen(channel)
	return result, errors.Wrap(err, "listening to channel")
}
