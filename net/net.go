package net

import (
	"net"
	"time"

	"chain/net/etcdname"
)

var knownNetworks = map[string]bool{
	"ip":   true,
	"ip4":  true,
	"ip6":  true,
	"udp":  true,
	"udp4": true,
	"udp6": true,
	"tcp":  true,
	"tcp4": true,
	"tcp6": true,
}

// LookupHost looks up the given host using etcd or the local resolver.
// It returns an array of that host's addresses
// See LookupHost in packages chain/net/etcdname and net for more.
func LookupHost(host string) (addrs []string, err error) {
	addrs, _, err = lookupHost(host)
	return addrs, err
}

// like LookupHost, but also returns the data source.
func lookupHost(host string) (addrs []string, source string, err error) {
	// Make sure empty hosts are rejected. See comment in
	// $GOROOT/src/net/lookup.go for more.
	if host == "" {
		return nil, "", &net.DNSError{Err: "no such host", Name: host}
	}

	if ip := net.ParseIP(host); ip != nil {
		return []string{host}, "literal", nil
	}

	if addrs, err := etcdname.LookupHost(host); err == nil {
		return addrs, "etcd", nil
	}

	addrs, err = net.LookupHost(host)
	return addrs, "stdlib", err
}

// Dialer satisfies the interface pq.Dialer, using chain/net/etcdname for
// hostname lookups in addition to the local resolver.
type Dialer net.Dialer

// DialTimeout acts like Dial but takes a timeout. The timeout includes name resolution, if required.
func (d *Dialer) DialTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	d1 := new(Dialer)
	*d1 = *d
	d1.Timeout = timeout
	return d1.Dial(network, addr)
}

// Dial connects to the address on the named network. Hostname lookup for addr
// uses package etcdname in addition to the local resolver.
// See func Dial for a description of the network and addr parameters.
func (d *Dialer) Dial(network, addr string) (net.Conn, error) {
	if knownNetworks[network] {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		addrs, source, err := lookupHost(host)
		if err != nil {
			return nil, err
		}
		if source == "etcd" || len(addrs) == 1 {
			// net.Dial iterates through the list of all possible addrs, but we don't need to yet.
			addr = net.JoinHostPort(addrs[0], port)
		}

		// Note: If addr didn't come from etcd AND if there were multiple addrs, this will do
		// another name lookup inside of net.DialTimeout because addr is still the original value.
	}
	return (*net.Dialer)(d).Dial(network, addr)
}
