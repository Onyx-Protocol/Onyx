package main

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"os"
	"time"

	"github.com/kr/secureheader"

	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/crypto/hsm/thales/see"
	"chain/crypto/hsm/thales/xprvseeclient"
	"chain/env"
	"chain/errors"
	"chain/log"
	"chain/log/rotation"
	"chain/log/splunk"
	"chain/net/http/authn"
	"chain/net/http/gzip"
	"chain/net/http/httpjson"
	"chain/net/http/reqid"
)

var (
	tlsCrt     = env.String("TLSCRT", "")
	tlsKey     = env.String("TLSKEY", "")
	listenAddr = env.String("LISTEN", ":8080")
	target     = env.String("TARGET", "sandbox")
	keyident   = env.String("KEY_IDENT", "dbgxprv1")
	userdata   = env.String("HSM_USERDATA", os.Getenv("CHAIN")+"/crypto/hsm/thales/xprvseemodule/userdata.sar")
	splunkAddr = os.Getenv("SPLUNKADDR")
	logFile    = os.Getenv("LOGFILE")
	logSize    = env.Int("LOGSIZE", 5e6) // 5MB
	logCount   = env.Int("LOGCOUNT", 9)
	password   = []byte(os.Getenv("PASSWORD"))

	// build vars; initialized by the linker
	buildTag    = "dev"
	buildCommit = "?"
	buildDate   = "?"

	race []interface{} // initialized in race.go
)

var (
	seeConn *see.Conn
	client  *xprvseeclient.Client

	xpubstr string
	kd      uint32
)

func main() {
	env.Parse()
	ctx := context.Background()

	stdlog.SetPrefix("signerd-" + buildTag + ": ")
	stdlog.SetFlags(stdlog.Lshortfile)
	log.SetPrefix(append(race, "app", "signerd", "target", *target, "buildtag", buildTag)...)
	log.SetOutput(logWriter())

	var err error
	seeConn, err = see.Open(*userdata)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}

	client = xprvseeclient.New(seeConn)

	err = loadKey(ctx)
	if err != nil {
		log.Fatal(ctx, log.KeyError, errors.Wrap(err, "loading hsm kd"))
	}

	m := http.NewServeMux()
	signHandler, err := httpjson.Handler(signTemplates, writeHTTPError)
	if err != nil {
		log.Error(ctx, err)
	}
	m.Handle("/sign-transaction", signHandler)

	var h http.Handler = m
	h = gzip.Handler{Handler: h}
	h = authn.BasicHandler{Auth: auth, Next: h}
	h = reqid.Handler(h)
	http.Handle("/", h)
	http.HandleFunc("/health", func(http.ResponseWriter, *http.Request) {})
	secureheader.DefaultConfig.PermitClearLoopback = true

	server := &http.Server{
		Addr:    *listenAddr,
		Handler: secureheader.DefaultConfig,
	}
	log.Write(ctx, "at", "serve")
	if *tlsCrt != "" {
		cert, err := tls.X509KeyPair([]byte(*tlsCrt), []byte(*tlsKey))
		if err != nil {
			log.Fatal(ctx, log.KeyError, errors.Wrap(err, "parsing tls X509 key pair"))
		}

		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		err = server.ListenAndServeTLS("", "") // uses TLS certs from above
		if err != nil {
			log.Fatal(ctx, log.KeyError, errors.Wrap(err, "ListenAndServeTLS"))
		}
	} else {
		secureheader.DefaultConfig.HTTPSRedirect = false
		err = server.ListenAndServe()
		if err != nil {
			log.Fatal(ctx, log.KeyError, errors.Wrap(err, "ListenAndServe"))
		}
	}
}

func loadKey(ctx context.Context) error {
	xprvid, err := seeConn.LoadKey("custom", *keyident)
	if err != nil {
		return err
	}

	kd, err = client.LoadXprv(xprvid)
	if err != nil {
		return err
	}

	xpubBytes, err := client.DeriveXpub(kd, nil)
	if err != nil {
		return err
	}

	if len(xpubBytes) != 64 {
		return fmt.Errorf("xpub should have 64 bytes, got %d bytes", len(xpubBytes))
	}

	var xpub chainkd.XPub
	copy(xpub[:], xpubBytes)

	xpubstr = xpub.String()
	return nil
}

func writeHTTPError(ctx context.Context, w http.ResponseWriter, err error) {
	logHTTPError(ctx, err)
	body, info := errInfo(err)
	httpjson.Write(ctx, w, info.HTTPStatus, body)
}

func logHTTPError(ctx context.Context, err error) {
	_, info := errInfo(err)
	keyvals := []interface{}{
		"status", info.HTTPStatus,
		"chaincode", info.ChainCode,
		log.KeyError, err,
	}
	if info.HTTPStatus == 500 {
		keyvals = append(keyvals, log.KeyStack, errors.Stack(err))
	}
	log.Write(ctx, keyvals...)
}

func signTemplates(ctx context.Context, x struct {
	Txs   []*txbuilder.Template `json:"transactions"`
	XPubs []string              `json:"xpubs"`
}) interface{} {
	log.Write(ctx, "at", "signTemplates", "n", len(x.Txs))
	var resp []interface{}
	for _, tpl := range x.Txs {
		err := txbuilder.Sign(ctx, tpl, x.XPubs, clientSigner)
		if err != nil {
			logHTTPError(ctx, err)
			info, _ := errInfo(err)
			resp = append(resp, info)
		} else {
			resp = append(resp, tpl)
		}
	}
	return resp
}

func clientSigner(_ context.Context, _ string, path [][]byte, data [32]byte) ([]byte, error) {
	return client.XSign(kd, path, data)
}

// TODO(kr): more flexible/secure authentication (e.g. kerberos style)
func auth(req *http.Request) error {
	_, pw, _ := req.BasicAuth()
	if subtle.ConstantTimeCompare([]byte(pw), password) != 1 {
		return authn.ErrNotAuthenticated
	}
	return nil
}

func logWriter() io.Writer {
	dropmsg := []byte("\nlog data dropped\n")
	rotation := &errlog{w: rotation.Create(logFile, *logSize, *logCount)}
	splunk := &errlog{w: splunk.New(splunkAddr, dropmsg)}

	switch {
	case logFile != "" && splunkAddr != "":
		return io.MultiWriter(rotation, splunk)
	case logFile != "" && splunkAddr == "":
		return rotation
	case logFile == "" && splunkAddr != "":
		return splunk
	}
	return os.Stdout
}

type errlog struct {
	w io.Writer
	t time.Time // protected by chain/log mutex
}

func (w *errlog) Write(p []byte) (int, error) {
	// We don't want to ruin our performance
	// when there's a persistent error
	// writing to a log sink.
	// Print to stderr at most once per minute.
	_, err := w.w.Write(p)
	if err != nil && time.Since(w.t) > time.Minute {
		stdlog.Println("chain/log:", err)
		w.t = time.Now()
	}
	return len(p), nil // report success for the MultiWriter
}
