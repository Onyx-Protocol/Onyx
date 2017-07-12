package core

import (
	"context"
	"net"
	"net/url"
	"path"
	"strings"

	"chain/core/config"
	"chain/database/pg"
	"chain/database/sinkdb"
	"chain/errors"
	"chain/net/raft"
)

// Config provides access to Chain Core configuration options
// and their values.
//
// TODO(jackson): Remove the pg.DB argument when the PostgreSQL
// url is stored in a configuration option.
func Config(ctx context.Context, db pg.DB, sdb *sinkdb.DB) (*config.Options, error) {
	opts := config.New(sdb)

	equalFirst := func(a, b []string) bool { return a[0] == b[0] }
	cleanEnclaveTuple := func(tup []string) error {
		normalized, err := normalizeURL(tup[0])
		if err != nil {
			return errors.WithDetailf(err, "Provided URL is invalid: %s", err.Error())
		}
		if normalized.Scheme != "https" {
			return errors.WithDetailf(err, "Enclave URL must use https.")
		}
		tup[0] = normalized.String()
		if !strings.Contains(tup[1], ":") {
			return errors.WithDetailf(err, "Access token must be of the form <username>:<password>.")
		}
		return nil
	}

	// enclave defines a set of (URL, access token) tuples to be used
	// for the local block signer. Tuple equality is defined on the
	// the URL, not the access token.
	opts.DefineSet("enclave", 2, cleanEnclaveTuple, equalFirst)

	// migrate any old-style existing configuration options
	monolith, err := config.Load(ctx, db, sdb)
	if errors.Root(err) == raft.ErrUninitialized {
		return opts, nil
	} else if err != nil {
		return nil, err
	}

	if monolith != nil {
		var ops []sinkdb.Op
		if monolith.BlockHsmUrl != "" {
			tup := []string{monolith.BlockHsmUrl, monolith.BlockHsmAccessToken}
			ops = append(ops, opts.Add("enclave", tup))
		}
		err = sdb.Exec(ctx, ops...)
		if err != nil {
			return nil, errors.Wrap(err, "migrating config options")
		}
	}
	return opts, nil
}

// normalizeURL performs some low-hanging best-effort normalization
// of the provided URL. See RFC3986, Section 6.
func normalizeURL(urlstr string) (*url.URL, error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return u, err
	}

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

	// Clean the path
	u.Path = path.Clean("/" + u.Path)

	return u, nil
}
