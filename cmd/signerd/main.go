package main

import (
	"crypto/subtle"
	"crypto/tls"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"os"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/kr/secureheader"
	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"

	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/crypto/hsm/thales/see"
	"chain/crypto/hsm/thales/xprvseeclient"
	"chain/env"
	"chain/errors"
	"chain/log"
	"chain/log/rotation"
	"chain/log/splunk"
	"chain/metrics"
	chainhttp "chain/net/http"
	"chain/net/http/authn"
	"chain/net/http/gzip"
	"chain/net/http/httpjson"
	"chain/net/http/httpspan"
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

	xpub string
	kd   uint32
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
		log.Fatal(ctx, "error", err)
	}

	client = xprvseeclient.New(seeConn)

	err = loadKey(ctx)
	if err != nil {
		log.Fatal(ctx, "error", "loading hsm kd: "+err.Error())
	}

	m := httpjson.NewServeMux(writeHTTPError)
	m.HandleFunc("POST", "/v1/signtemplates", signTemplates)

	var h chainhttp.Handler = m
	h = metrics.Handler{Handler: h}
	h = gzip.Handler{Handler: h}
	h = httpspan.Handler{Handler: h}
	h = authn.BasicHandler{Auth: auth, Realm: "signerd", Next: h}
	http.Handle("/", chainhttp.ContextHandler{Context: ctx, Handler: h})
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
			log.Fatal(ctx, "error", "parsing tls X509 key pair: "+err.Error())
		}

		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		err = server.ListenAndServeTLS("", "") // uses TLS certs from above
	} else {
		secureheader.DefaultConfig.HTTPSRedirect = false
		err = server.ListenAndServe()
	}
	if err != nil {
		log.Fatal(ctx, "error", "ListenAndServe: "+err.Error())
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

	if len(xpubBytes) == 78 {
		checkSum := sha3.Sum256(xpubBytes)
		xpubBytes = append(xpubBytes, checkSum[:4]...)
	}

	if len(xpubBytes) != 82 {
		return fmt.Errorf("xpub should have 82 bytes, got %d bytes", len(xpubBytes))
	}

	xpub = base58.Encode(xpubBytes)
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

func signTemplates(ctx context.Context, txs []*txbuilder.Template) interface{} {
	log.Write(ctx, "at", "signTemplates", "n", len(txs))
	var resp []interface{}
	for _, tpl := range txs {
		err := signTemplate(ctx, tpl)
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

func signTemplate(ctx context.Context, tpl *txbuilder.Template) error {
	txbuilder.ComputeSigHashes(ctx, tpl) // don't trust the sighashes in the request
	// TODO(kr): come up with some scheme to verify that the
	// covered output scripts are what the client really wants.
	for i, input := range tpl.Inputs {
		if len(input.SigComponents) > 0 {
			for c, component := range input.SigComponents {
				for s, sig := range component.Signatures {
					if sig.XPub == xpub {
						sigdata, err := client.Sign(kd, sig.DerivationPath, component.SignatureData)
						if err != nil {
							return errors.Wrapf(err, "computing signature for input %d, sigscript component %d, sig %d", i, c, s)
						}
						sig.Bytes = append(sigdata, byte(bc.SigHashAll))
					}
				}
			}
		}
	}
	return nil
}

// TODO(kr): more flexible/secure authentication (e.g. kerberos style)
func auth(ctx context.Context, name, pw string) (authID string, err error) {
	if subtle.ConstantTimeCompare([]byte(pw), password) != 1 {
		return "", authn.ErrNotAuthenticated
	}
	return "user", nil
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
