//+build localhost_auth

package main

import "chain/core/config"

// See $CHAIN/net/http/authz/localhost_auth.go for the implementation.
func init() {
	config.BuildConfig.LocalhostAuth = true
}
