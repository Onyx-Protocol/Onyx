package core

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"

	"chain/errors"
)

var ErrNoTLS = errors.New("no TLS configuration available")

// TLSConfig returns a TLS config suitable for use
// as a Chain Core client and server.
// It reads a PEM-encoded X.509 certificate and private key
// from certFile and keyFile.
// If rootCAs is given, it reads a list of trusted root CA certs
// from the filesystem;
// otherwise it uses the system cert pool.
//
// For compatibility, it attempts to read the cert and key
// from the environment if certFile and keyFile both
// do not exist in the filesystem.
//   TLSCRT=[PEM-encoded X.509 certificate]
//   TLSKEY=[PEM-encoded X.509 private key]
//
// If certFile and keyFile do not exist or are empty
// and the environment vars are both unset,
// TLSConfig returns ErrNoTLS.
func TLSConfig(certFile, keyFile, rootCAs string) (*tls.Config, error) {
	config := &tls.Config{
		// This is the default set of protocols for package http.
		// ListenAndServeTLS and Transport set this automatically,
		// but since we're supplying our own TLS config,
		// we have to set it here.
		NextProtos: []string{"http/1.1", "h2"},
		ClientAuth: tls.VerifyClientCertIfGiven,
	}

	cert, certErr := ioutil.ReadFile(certFile)
	key, keyErr := ioutil.ReadFile(keyFile)
	if os.IsNotExist(certErr) && os.IsNotExist(keyErr) {
		cert, key = []byte(os.Getenv("TLSCRT")), []byte(os.Getenv("TLSKEY"))
	} else if certErr != nil {
		return nil, certErr
	} else if keyErr != nil {
		return nil, keyErr
	}
	if len(cert) == 0 && len(key) == 0 {
		return nil, ErrNoTLS
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if rootCAs != "" {
		config.RootCAs, err = loadRootCAs(rootCAs)
	}
	config.ClientCAs = config.RootCAs
	return config, err
}

// loadRootCAs reads a list of PEM-encoded X.509 certificates from name
func loadRootCAs(name string) (*x509.CertPool, error) {
	pem, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(pem)
	if !ok {
		return nil, errors.Wrap(errors.New("cannot parse certs"))
	}
	return pool, nil
}
