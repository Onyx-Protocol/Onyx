//+build http_ok

package main

import "chain/core/config"

/*
This file exposes a build tag to permit plaintext (non-TLS) HTTP.
TLS will be required by default. Disabling TLS will be useful for
chain core developer edition. Users will be able to connect to a
chain core without a needing a TLS cert.
*/

func init() {
	config.BuildConfig.HTTPOk = true
}
