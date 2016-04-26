package net

import (
	"net"

	"chain/net/etcdname"
)

// LookupHost looks up the given host using etcd or the local resolver.
// It returns an array of that host's addresses
// See LookupHost in packages chain/net/etcdname and net for more.
func LookupHost(host string) (addrs []string, err error) {
	// Make sure empty hosts are rejected. See comment in
	// $GOROOT/src/net/lookup.go for more.
	if host == "" {
		return nil, &net.DNSError{Err: "no such host", Name: host}
	}

	if ip := net.ParseIP(host); ip != nil {
		return []string{host}, nil
	}

	if addrs, err := etcdname.LookupHost(host); err == nil {
		return addrs, nil
	}

	return net.LookupHost(host)
}
