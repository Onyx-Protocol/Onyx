//+build unauthn_loopback

package main

import (
	"net"
	"net/http"
)

func unauthnLoopback(req *http.Request) bool {
	// Allow connections from the local host.
	a, err := net.ResolveTCPAddr("tcp", req.RemoteAddr)
	return err == nil && a.IP.IsLoopback()
}
