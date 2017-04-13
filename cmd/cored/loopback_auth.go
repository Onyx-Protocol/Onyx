//+build loopback_auth

package main

import "chain/core/config"

// See $CHAIN/net/http/authz/loopback_authz.go for the implementation.
func init() {
	config.BuildConfig.LoopbackAuth = true
}
