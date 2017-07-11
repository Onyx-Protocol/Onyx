package core

import (
	"net"
	"net/url"
	"path"
	"strings"

	"chain/core/config"
	"chain/database/sinkdb"
	"chain/errors"
)

// Config provides access to Chain Core configuration options
// and their values.
func Config(sdb *sinkdb.DB) *config.Options {
	opts := config.New(sdb)

	equalFirst := func(a, b []string) bool { return a[0] == b[0] }
	cleanEnclaveTuple := func(tup []string) error {
		normalized, err := normalizeURL(tup[0])
		if err != nil {
			return errors.WithDetailf(err, "Provided URL is invalid: %s", err.Error())
		}
		tup[0] = normalized
		return nil
	}

	// enclave defines a set of (URL, access token) tuples to be used
	// for the local block signer. Tuple equality is defined on the
	// the URL, not the access token.
	opts.DefineSet("enclave", 2, cleanEnclaveTuple, equalFirst)

	return opts
}

// normalizeURL performs some low-hanging best-effort normalization
// of the provided URL. See RFC3986, Section 6.
func normalizeURL(urlstr string) (string, error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return "", err
	}

	u.Path = path.Clean(u.Path)
	// Lowercase case-insensitive portions
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	// TODO(jackson): perform IDNA host ToASCII conversion?

	// Remove empty or default port numbers.
	host, port, err := net.SplitHostPort(u.Host)
	if err == nil {
		switch {
		case port == "":
			// a port separator without a port; just use the
			// hostname
			u.Host = host
		case port == "80" && u.Scheme == "http":
			// port 80 is already the default port for http
			u.Host = host
		case port == "443" && u.Scheme == "https":
			// port 443 is already the default port for https
			u.Host = host
		}
	}
	return u.String(), nil
}
