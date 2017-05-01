package raft

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"

	"chain/errors"
)

func verifyTLSName(addr string, client *http.Client) error {
	hostname, _, _ := net.SplitHostPort(addr)
	c := clientTLS(client)
	if c == nil {
		return nil
	}
	x509Cert, err := x509.ParseCertificate(c.Certificates[0].Certificate[0])
	if err != nil {
		return errors.Wrap(err)
	}
	return errors.Wrap(x509Cert.VerifyHostname(hostname))
}

func clientTLS(c *http.Client) *tls.Config {
	t, ok := c.Transport.(*http.Transport)
	if !ok {
		return nil
	}
	return t.TLSClientConfig
}
