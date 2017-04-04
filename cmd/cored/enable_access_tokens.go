//+build enable_access_tokens

package main

import (
	"chain/core/config"
	"net/http"
)

func init() {
	config.BuildConfig.AccessTokens = true
}

func authLoopbackInDev(req *http.Request) bool {
	return false
}
