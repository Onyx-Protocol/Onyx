package main

import (
	"chain/core/fileutil"
	"chain/errors"
	"chain/log"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	certsDir            = filepath.Join(fileutil.DefaultDir(), "certs") + string(filepath.Separator)
	certFileExt         = getCertFileExt()
	defaultCertDuration = 10 * 365 * 24 * time.Hour
	defaultCATemplate   = &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:               true,
		KeyUsage:           x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		SignatureAlgorithm: x509.SHA256WithRSA,
		Subject: pkix.Name{
			CommonName:         "Chain Core Developer Edition Mock CA",
			Organization:       []string{"Chain"},
			OrganizationalUnit: []string{"Engineering"},
			Locality:           []string{"San Francisco"},
			Country:            []string{"US"},
		},
		NotBefore:    notBefore(),
		NotAfter:     notBefore().Add(defaultCertDuration).UTC(),
		SerialNumber: generateSerialNumber(),
	}
	defaultCertTemplate = &x509.Certificate{
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "cored.dev"},
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		Subject: pkix.Name{
			CommonName:         "localhost",
			Organization:       []string{"Chain"},
			OrganizationalUnit: []string{"Engineering"},
			Locality:           []string{"San Francisco"},
			Country:            []string{"US"},
		},
		NotBefore:    notBefore(),
		NotAfter:     notBefore().Add(defaultCertDuration).UTC(),
		SerialNumber: generateSerialNumber(),
	}
)

// generatePKIX checks if a development pkix
// exists on the host and generates one if necessary.
func generatePKIX(ctx context.Context, serverCertPath, serverKeyPath, caPath *string) error {
	*caPath = certsDir + "ca" + certFileExt
	exists, err := exist(*caPath)
	if err != nil {
		return err
	}

	*serverCertPath = certsDir + "server" + certFileExt
	*serverKeyPath = certsDir + "server.key"
	*caPath = certsDir + "ca" + certFileExt
	if exists {
		return warn()
	}

	err = os.MkdirAll(certsDir, 0755)
	if err != nil {
		return errors.Wrap(err, "generating development pkix directory")
	}

	ca, key, err := generatePEMKeyPair("ca", defaultCATemplate, nil, 2048, nil)
	if err != nil {
		return errors.Wrap(err, "generating root ca keypair")
	}

	caCert, caKey, err := parsePEMKeypair(ca, key)
	if err != nil {
		return errors.Wrap(err, "parsing root ca keypair")
	}

	_, _, err = generatePEMKeyPair("server", defaultCertTemplate, caCert, 2048, caKey)
	if err != nil {
		return errors.Wrap(err, "generating server keypair")
	}

	_, _, err = generatePEMKeyPair("client", defaultCertTemplate, caCert, 2048, caKey)
	if err != nil {
		return errors.Wrap(err, "generating server keypair")
	}
	return warn()
}

func generatePEMKeyPair(name string, req, ca *x509.Certificate, keySize int, priv *rsa.PrivateKey) ([]byte, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating private key")
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	keyBytes = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})

	if ca == nil {
		ca = req
	}
	if priv == nil {
		priv = key
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, req, ca, &key.PublicKey, priv)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating certificate")
	}
	certBytes = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	err = writeKeyPair(certBytes, keyBytes, certsDir+name+certFileExt, certsDir+name+".key")
	if err != nil {
		return nil, nil, errors.Wrap(err, "writing keypair")
	}
	return certBytes, keyBytes, nil
}

func parsePEMKeypair(c, k []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	block, _ := pem.Decode(c)
	if block == nil {
		return nil, nil, errors.New("failed to parse certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, errors.New("parsing certificate")
	}

	block, _ = pem.Decode(k)
	if block == nil {
		return nil, nil, errors.New("failed to parse private key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, errors.New("parsing private key")
	}
	return cert, key, nil
}

func notBefore() time.Time {
	return time.Now().Add(-24 * time.Hour).UTC()
}

// Taken from https://github.com/cloudflare/cfssl/blob/master/signer/local/local.go
func generateSerialNumber() *big.Int {
	// RFC 5280 4.1.2.2:
	// Certificate users MUST be able to handle serialNumber
	// values up to 20 octets.  Conforming CAs MUST NOT use
	// serialNumber values longer than 20 octets.
	serialNumber := make([]byte, 20)
	_, err := io.ReadFull(rand.Reader, serialNumber)
	if err != nil {
		log.Fatalkv(context.Background(), log.KeyError, errors.New(fmt.Sprintf("failed to create certificate serial number: %v", err)))
	}

	// SetBytes interprets buf as the bytes of a big-endian
	// unsigned integer. The leading byte should be masked
	// off to ensure it isn't negative.
	serialNumber[0] &= 0x7F
	return new(big.Int).SetBytes(serialNumber)
}

func writeKeyPair(cBytes, kBytes []byte, cFile, kFile string) error {
	err := ioutil.WriteFile(cFile, cBytes, 0644)
	if err != nil {
		return errors.Wrap(err, "writing "+cFile)
	}

	err = ioutil.WriteFile(kFile, kBytes, 0644)
	if err != nil {
		return errors.Wrap(err, "writing "+kFile)
	}
	return nil
}

func exist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		if os.IsPermission(err) {
			return true, nil
		}
		return false, err
	}
	return true, nil
}

func wrapQuotes(str string) string {
	return "\"" + str + "\""
}

func warn() error {
	fmt.Printf("WARNING: This Chain Core is configured to require TLS. Development certificates and keys have been generated in %s\n\n", wrapQuotes(certsDir))
	switch runtime.GOOS {
	case "darwin":
		return warnDarwin()
	case "windows":
		return warnWindows()
	default:
		return warnDefault()
	}
}

func warnDarwin() error {
	// retrieve user's login keychain
	cmd := exec.Command("security", "login-keychain")
	out, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, "finding login keychain")
	}

	loginChain := strings.TrimSpace(string(out))
	installRoot := fmt.Sprintln("sudo", "security", "add-trusted-cert", "-d", "-r", "trustRoot -k", "/Library/Keychains/System.keychain", wrapQuotes(certsDir+"ca"+certFileExt))
	installCert := fmt.Sprintln("security", "import", wrapQuotes(certsDir+"client"+certFileExt), "-k", loginChain)
	installKey := fmt.Sprintln("security", "import", wrapQuotes(certsDir+"client.key"), "-k", loginChain)
	fmt.Printf("To install the root CA into the System Keychain run:\n\n\t%s\n", installRoot)
	fmt.Printf("To import the client cert into your login keychain run the following commands:\n\n\t%s\n\t%s\n", installCert, installKey)
	return nil
}

func warnWindows() error {
	installRoot := fmt.Sprintln("certutil", "-f", "-user", "-addstore", "Root", wrapQuotes(certsDir+"ca"+certFileExt))
	fmt.Printf("To install the root CA certificate into your user certificate store run:\n\n\t%s\n", installRoot)
	return nil
}

func warnDefault() (err error) {
	// TODO(boymanjor): detect OS and provide instructions on storing root CA
	fmt.Printf("You will need to install the root CA into your local certficate store to use the browser securely.\n")
	return nil
}

func getCertFileExt() string {
	if runtime.GOOS == "windows" {
		return ".cer"
	}
	return ".pem"
}
