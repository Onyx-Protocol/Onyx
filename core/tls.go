package core

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"

	"chain/errors"
	"chain/net"
)

var ErrNoTLS = errors.New("no TLS configuration available")

// TLSConfig returns a TLS config suitable for use
// as a Chain Core client and server.
// It reads a PEM-encoded X.509 certificate and private key
// from certFile and keyFile.
// If rootCAs is given,
// it should name a file containing a list of trusted root CA certs,
// otherwise the returned config uses the system cert pool.
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
	config := net.DefaultTLSConfig()

	// This is the default set of protocols for package http.
	// ListenAndServeTLS and Transport set this automatically,
	// but since we're supplying our own TLS config,
	// we have to set it here.
	// TODO(kr): disabled for now; consider adding h2 support here.
	// See also the comment on TLSNextProto in $CHAIN/cmd/cored/main.go.
	//NextProtos: []string{"http/1.1", "h2"},
	config.ClientAuth = tls.RequestClientCert

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

	config.RootCAs, err = loadRootCAs(rootCAs)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	// This TLS config is used by cored peers to dial each other,
	// and by corectl to dial cored.
	// All those processes have the same identity,
	// so we automatically trust the local cert,
	// with the expectation that the peer will also be using it.
	// This makes misconfiguation impossible.
	// (For some reason, X509KeyPair doesn't keep a copy of the leaf cert,
	// so we need to parse it again here.)
	x509Cert, err := x509.ParseCertificate(config.Certificates[0].Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err)
	}
	config.RootCAs.AddCert(x509Cert)
	config.ClientCAs = config.RootCAs
	return config, err
}

// loadRootCAs reads a list of PEM-encoded X.509 certificates from name.
// If name is the empty string, it returns a new, empty cert pool.
func loadRootCAs(name string) (*x509.CertPool, error) {
	if name == "" {
		return x509.NewCertPool(), nil
	}
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
