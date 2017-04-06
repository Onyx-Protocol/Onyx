//+build no_loopback_authn

package main

import (
	"chain/core/config"
	"net"
	"net/http"
)

func init() {
	config.BuildConfig.LoopbackAuthn = false
	unauthnLoopback = func(req *http.Request) bool {
		// Allow connections from the local host.
		a, err := net.ResolveTCPAddr("tcp", req.RemoteAddr)
		return err == nil && a.IP.IsLoopback()
	}
}
