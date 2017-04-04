//+build enable_access_tokens

package main

import "net/http"

func init() {
	config.AccessTokens = true
}

func authLoopbackInDev(req *http.Request) bool {
	return false
}
