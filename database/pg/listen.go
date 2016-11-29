package pg

import (
	"context"
	"time"

	"github.com/lib/pq"

	"chain-stealth/errors"
	"chain-stealth/log"
	"chain-stealth/net"
)

// NewListener creates a new pq.Listener and begins listening.
func NewListener(ctx context.Context, dbURL, channel string) (*pq.Listener, error) {
	// We want etcd name lookups so we use our own Dialer.
	d := new(net.Dialer)
	result := pq.NewDialListener(d, dbURL, 1*time.Second, 10*time.Second, func(ev pq.ListenerEventType, err error) {
		log.Error(ctx, errors.Wrapf(err, "event in %s listener: %v", channel, ev))
	})
	err := result.Listen(channel)
	return result, errors.Wrap(err, "listening to channel")
}
