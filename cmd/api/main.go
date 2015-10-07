package main

import (
	"database/sql"
	"log"
	"net/http"
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
	tlsCrt       = env.String("TLSCRT", "")
	tlsKey       = env.String("TLSKEY", "")
	listenAddr   = env.String("LISTEN", ":8080")
	dbURL        = env.String("DB_URL", "postgres:///api?sslmode=disable")
	stack        = env.String("STACK", "sandbox")
	samplePer    = env.Duration("SAMPLEPER", 10*time.Second)
	nouserSecret = env.String("NOUSER_SECRET", "")

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
		librato.Source = *stack
		go librato.SampleMetrics(*samplePer)
	} else {
		log.Println("no metrics; set LIBRATO_URL for prod")
	}

	var err error
	db, err = sql.Open("schemadb", *dbURL)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(100)
	appdb.Init(db)

	var h chainhttp.Handler
	h = api.Handler(*nouserSecret)
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
