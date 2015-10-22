package main

import (
	"database/sql"
	"expvar"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/kr/env"
	"github.com/kr/secureheader"
	"golang.org/x/net/context"

	"chain/api"
	"chain/api/appdb"
	"chain/database/pg"
	chainlog "chain/log"
	"chain/log/rotation"
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
	logFile      = os.Getenv("LOGFILE")
	logSize      = env.Int("LOGSIZE", 5e6) // 5MB
	logCount     = env.Int("LOGCOUNT", 9)
	// for config var LIBRATO_URL, see func init below

	// build vars; initialized by the linker
	buildTag    = "dev"
	buildCommit = "?"
	buildDate   = "?"
)

func init() {
	librato.URL = env.URL("LIBRATO_URL", "")
	librato.Prefix = "chain.api."
	expvar.NewString("buildtag").Set(buildTag)
	expvar.NewString("builddate").Set(buildDate)
	expvar.NewString("buildcommit").Set(buildCommit)
}

func main() {
	env.Parse()
	sql.Register("schemadb", pg.SchemaDriver(buildTag))
	log.SetPrefix("api-" + buildTag + ": ")
	log.SetFlags(log.Lshortfile)
	if logFile != "" {
		log.SetOutput(rotation.Create(logFile+".stdlib", *logSize, *logCount))
		chainlog.SetOutput(rotation.Create(logFile, *logSize, *logCount))
	}

	if librato.URL.Host != "" {
		librato.Source = *stack
		go librato.SampleMetrics(*samplePer)
	} else {
		log.Println("no metrics; set LIBRATO_URL for prod")
	}

	db, err := sql.Open("schemadb", *dbURL)
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
