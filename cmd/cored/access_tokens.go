//+build !enable_access_tokens

package main

import (
	"chain/core/config"
	"net"
	"net/http"
)

func init() {
	config.AccessTokens = false
}

func authLoopbackInDev(req *http.Request) bool {
	// Allow connections from the local host.
	a, err := net.ResolveTCPAddr("tcp", req.RemoteAddr)
	return err == nil && a.IP.IsLoopback()
}
