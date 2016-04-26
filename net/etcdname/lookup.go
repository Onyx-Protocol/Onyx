package etcdname

import "errors"

// TODO: configure etcd cluster, define protocol for name lookups

// LookupHost looks up the given host using the configured etcd cluster, if any.
// It returns an array of that host's addresses.
func LookupHost(host string) (addrs []string, err error) {
	return nil, errors.New("unimplemented")
}
