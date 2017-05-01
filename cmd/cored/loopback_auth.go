//+build loopback_auth
// TODO: for consistent language, rename this tag to localhost_auth

package main

import "chain/core/config"

// See $CHAIN/net/http/authz/loopback_authz.go for the implementation.
func init() {
	config.BuildConfig.LocalhostAuth = true
}
