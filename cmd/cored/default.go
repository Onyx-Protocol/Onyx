//+build !unauthn_loopback

package main

import "net/http"

func unauthnLoopback(req *http.Request) bool {
	return false
}
