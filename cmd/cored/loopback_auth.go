//+build loopback_auth

package main

import "chain/core/config"

func init() {
	config.BuildConfig.LoopbackAuth = true
}
