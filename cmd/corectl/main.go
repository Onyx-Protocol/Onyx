package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"chain/core/accesstoken"
	"chain/core/config"
	"chain/core/migrate"
	"chain/crypto/ed25519"
	"chain/database/sql"
	chainjson "chain/encoding/json"
	"chain/env"
	"chain/generated/rev"
	"chain/log"
)

// config vars
var (
	dbURL = env.String("DATABASE_URL", "postgres:///core?sslmode=disable")

	// build vars; initialized by the linker
	buildTag    = "?"
	buildCommit = "?"
	buildDate   = "?"
)

// We collect log output in this buffer,
// and display it only when there's an error.
var logbuf bytes.Buffer

type command struct {
	f func(*sql.DB, []string)
}

var commands = map[string]*command{
	"config-generator":     {configGenerator},
	"create-block-keypair": {createBlockKeyPair},
	"create-token":         {createToken},
	"config":               {configNongenerator},
	"migrate":              {runMigrations},
	"reset":                {reset},
}

func main() {
	log.SetOutput(&logbuf)
	env.Parse()

	if len(os.Args) >= 2 && os.Args[1] == "-version" {
		var version string
		if buildTag != "?" {
			// build tag with chain-core-server- prefix indicates official release
			version = strings.TrimPrefix(buildTag, "chain-core-server-")
		} else {
			// version of the form rev123 indicates non-release build
			version = rev.ID
		}
		fmt.Printf("corectl (Chain Core) %s\n", version)
		versionProdPrintln()
		fmt.Printf("build-commit: %v\n", buildCommit)
		fmt.Printf("build-date: %v\n", buildDate)
		return
	}

	db, err := sql.Open("hapg", *dbURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	if len(os.Args) < 2 {
		help(os.Stdout)
		os.Exit(0)
	}
	cmd := commands[os.Args[1]]
	if cmd == nil {
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		help(os.Stderr)
		os.Exit(1)
	}
	cmd.f(db, os.Args[2:])
}

func runMigrations(db *sql.DB, args []string) {
	const usage = "usage: corectl migrate [-status]"

	var flags flag.FlagSet
	flagStatus := flags.Bool("status", false, "print all migrations and their status")
	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	if len(flags.Args()) != 0 {
		fatalln("error: migrate takes no args")
	}

	var err error
	if *flagStatus {
		err = migrate.PrintStatus(db)
	} else {
		err = migrate.Run(db)
	}
	if err != nil {
		fatalln("error: ", err)
	}
}

func configGenerator(db *sql.DB, args []string) {
	const usage = "usage: corectl config-generator [flags] [quorum] [pubkey url]..."
	var (
		quorum  int
		signers []config.BlockSigner
		err     error
	)

	var flags flag.FlagSet
	maxIssuanceWindow := flags.Duration("w", 24*time.Hour, "the maximum issuance window `duration` for this generator")
	flagK := flags.String("k", "", "local `pubkey` for signing blocks")
	flagHSMURL := flags.String("hsm-url", "", "hsm `url` for signing blocks (mockhsm if empty)")
	flagHSMToken := flags.String("hsm-token", "", "hsm `access-token` for connecting to hsm")

	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()

	// not a blocksigner
	if *flagK == "" && *flagHSMURL != "" {
		fatalln("error: flag -hsm-url has no effect without -k")
	}

	// TODO(ameets): update when switching to x.509 authorization
	if (*flagHSMURL == "") != (*flagHSMToken == "") {
		fatalln("error: flags -hsm-url and -hsm-token must be given together")
	}

	if len(args) == 0 {
		if *flagK != "" {
			quorum = 1
		}
	} else if len(args)%2 != 1 {
		fatalln(usage)
	} else {
		quorum, err = strconv.Atoi(args[0])
		if err != nil {
			fatalln(usage)
		}

		for i := 1; i < len(args); i += 2 {
			pubkey, err := hex.DecodeString(args[i])
			if err != nil {
				fatalln(usage)
			}
			if len(pubkey) != ed25519.PublicKeySize {
				fatalln("error:", "bad ed25519 public key length")
			}
			url := args[i+1]
			signers = append(signers, config.BlockSigner{
				Pubkey: pubkey,
				URL:    url,
			})
		}
	}

	conf := &config.Config{
		IsGenerator: true,
		Quorum:      quorum,
		Signers:     signers,
		MaxIssuanceWindow: chainjson.Duration{
			Duration: *maxIssuanceWindow,
		},
		IsSigner:            *flagK != "",
		BlockPub:            *flagK,
		BlockHSMURL:         *flagHSMURL,
		BlockHSMAccessToken: *flagHSMToken,
	}

	ctx := context.Background()
	migrateIfMissingSchema(ctx, db)
	err = config.Configure(ctx, db, conf)
	if err != nil {
		fatalln("error:", err)
	}

	fmt.Println("blockchain id", conf.BlockchainID)
}

func createToken(db *sql.DB, args []string) {
	const usage = "usage: corectl create-token [-net] [name]"
	var flags flag.FlagSet
	flagNet := flags.Bool("net", false, "create a network token instead of client")
	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()
	if len(args) < 1 {
		fatalln(usage)
	}

	ctx := context.Background()
	migrateIfMissingSchema(ctx, db)
	accessTokens := &accesstoken.CredentialStore{DB: db}
	typ := map[bool]string{true: "network", false: "client"}[*flagNet]
	tok, err := accessTokens.Create(ctx, args[0], typ)
	if err != nil {
		fatalln("error:", err)
	}
	fmt.Println(tok.Token)
}

func configNongenerator(db *sql.DB, args []string) {
	const usage = "usage: corectl config [flags] [blockchain-id] [generator-url]"
	var flags flag.FlagSet
	flagT := flags.String("t", "", "generator access `token`")
	flagK := flags.String("k", "", "local `pubkey` for signing blocks")
	flagHSMURL := flags.String("hsm-url", "", "hsm `url` for signing blocks (mockhsm if empty)")
	flagHSMToken := flags.String("hsm-token", "", "hsm `access-token` for connecting to hsm")

	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()
	if len(args) < 2 {
		fatalln(usage)
	}

	// not a blocksigner
	if *flagK == "" && *flagHSMURL != "" {
		fatalln("error: flag -hsm-url has no effect without -k")
	}

	// TODO(ameets): update when switching to x.509 authorization
	if (*flagHSMURL == "") != (*flagHSMToken == "") {
		fatalln("error: flags -hsm-url and -hsm-token must be given together")
	}

	var conf config.Config
	err := conf.BlockchainID.UnmarshalText([]byte(args[0]))
	if err != nil {
		fatalln("error: invalid blockchain ID:", err)
	}
	conf.GeneratorURL = args[1]
	conf.GeneratorAccessToken = *flagT
	conf.IsSigner = *flagK != ""
	conf.BlockPub = *flagK
	conf.BlockHSMURL = *flagHSMURL
	conf.BlockHSMAccessToken = *flagHSMToken

	ctx := context.Background()
	migrateIfMissingSchema(ctx, db)
	err = config.Configure(ctx, db, &conf)
	if err != nil {
		fatalln("error:", err)
	}
}

// migrateIfMissingSchema will migrate the provided database only
// if the database is blank without any migrations.
func migrateIfMissingSchema(ctx context.Context, db *sql.DB) {
	const q = `SELECT to_regclass('migrations') IS NOT NULL`
	var initialized bool
	err := db.QueryRow(ctx, q).Scan(&initialized)
	if err != nil {
		fatalln("initializing schema", err)
	}
	if initialized {
		return
	}

	err = migrate.Run(db)
	if err != nil {
		fatalln("initializing schema", err)
	}
}

func fatalln(v ...interface{}) {
	io.Copy(os.Stderr, &logbuf)
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(2)
}

func help(w io.Writer) {
	fmt.Fprintln(w, "usage: corectl [-version] [command] [arguments]")
	fmt.Fprint(w, "\nThe commands are:\n\n")
	for name := range commands {
		fmt.Fprintln(w, "\t", name)
	}
	fmt.Fprint(w, "\nFlags:\n")
	fmt.Fprintln(w, "\t-version   print version information")
	fmt.Fprintln(w)
}
