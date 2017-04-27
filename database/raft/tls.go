package raft

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"chain/errors"
)

func verifyTLSName(name string, client *http.Client) error {
	c := clientTLS(client)
	if c == nil {
		return nil
	}
	x509Cert, err := x509.ParseCertificate(c.Certificates[0].Certificate[0])
	if err != nil {
		return errors.Wrap(err)
	}
	return errors.Wrap(x509Cert.VerifyHostname(name))
}

func clientTLS(c *http.Client) *tls.Config {
	t, ok := c.Transport.(*http.Transport)
	if !ok {
		return nil
	}
	return t.TLSClientConfig
}
