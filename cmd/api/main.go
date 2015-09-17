package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kr/env"
	"github.com/kr/secureheader"
	"golang.org/x/net/context"

	"chain/api"
	"chain/api/appdb"
	"chain/database/pg"
	"chain/metrics"
	"chain/metrics/librato"
	chainhttp "chain/net/http"
	"chain/net/http/gzip"
)

var (
	// config vars
	tlsCrt     = env.String("TLSCRT", "")
	tlsKey     = env.String("TLSKEY", "")
	listenAddr = env.String("LISTEN", ":8080")
	dbURL      = env.String("DB_URL", "postgres:///api?sslmode=disable")
	samplePer  = env.Duration("SAMPLEPER", 10*time.Second)

	db       *sql.DB
	buildTag = "dev"
)

func main() {
	librato.URL = env.URL("LIBRATO_URL", "")
	librato.Prefix = "chain.api."

	sql.Register("schemadb", pg.SchemaDriver(buildTag))
	log.SetPrefix("api-" + buildTag + ": ")
	log.SetFlags(log.Lshortfile)
	env.Parse()

	if librato.URL.Host != "" {
		librato.Source, _ = os.Hostname()
		go librato.SampleMetrics(*samplePer)
	} else {
		log.Println("no metrics; set LIBRATO_URL for prod")
	}

	var err error
	db, err = sql.Open("schemadb", *dbURL)
	if err != nil {
		log.Fatal(err)
	}
	appdb.Init(db)

	var h chainhttp.Handler
	h = api.Handler() // TODO(kr): authentication
	h = metrics.Handler{Handler: h}
	h = gzip.Handler{Handler: h}

	bg := context.Background()
	bg = pg.NewContext(bg, db)
	http.Handle("/", chainhttp.ContextHandler{Context: bg, Handler: h})
	http.HandleFunc("/health", func(http.ResponseWriter, *http.Request) {})

	secureheader.DefaultConfig.PermitClearLoopback = true

	if *tlsCrt != "" {
		err = chainhttp.ListenAndServeTLS(*listenAddr, *tlsCrt, *tlsKey, secureheader.DefaultConfig)
	} else {
		err = http.ListenAndServe(*listenAddr, secureheader.DefaultConfig)
	}
	if err != nil {
		log.Fatalln("ListenAndServe:", err)
	}
}
