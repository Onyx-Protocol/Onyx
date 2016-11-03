package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"chain/core/accesstoken"
	"chain/core/config"
	"chain/core/migrate"
	"chain/core/mockhsm"
	"chain/crypto/ed25519"
	"chain/database/raft"
	"chain/database/sql"
	"chain/env"
	"chain/log"
	"chain/protocol/bc"
)

// config vars
var (
	dbURL      = env.String("DATABASE_URL", "postgres:///core?sslmode=disable")
	listenAddr = env.String("LISTEN", ":1999")
	dir        = env.String("DIR", defaultDir())
	bootURL    = env.String("BOOTURL", "")
)

// We collect log output in this buffer,
// and display it only when there's an error.
var logbuf bytes.Buffer

type command struct {
	f func(*sql.DB, *raft.Service, []string)
}

var commands = map[string]*command{
	"config-generator":     {configGenerator},
	"create-block-keypair": {createBlockKeyPair},
	"create-token":         {createToken},
	"config":               {configNongenerator},
	"reset":                {reset},
}

func main() {
	log.SetOutput(&logbuf)
	env.Parse()
	db, err := sql.Open("hapg", *dbURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	raftDir := filepath.Join(*dir, "raft") // TODO(kr): better name for this
	raftDB, err := raft.Start(*listenAddr, raftDir, *bootURL)
	if err != nil {
		fatalln("error: could not connect to raftDB", err)
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
	err = migrate.Run(db)
	if err != nil {
		fatalln("error: init schema", err)
	}
	cmd.f(db, raftDB, os.Args[2:])
}

func configGenerator(db *sql.DB, rDB *raft.Service, args []string) {
	const usage = "usage: corectl config-generator [-s] [-w duration] [quorum] [pubkey url]..."
	var (
		quorum  uint32
		signers []*config.BlockSigner
		err     error
	)

	var flags flag.FlagSet
	maxIssuanceWindow := flags.Duration("w", 24*time.Hour, "the maximum issuance window `duration` for this generator")
	isSigner := flags.Bool("s", false, "whether this core is a signer")
	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()

	if len(args) == 0 {
		if *isSigner {
			quorum = 1
		}
	} else if len(args)%2 != 1 {
		fatalln(usage)
	} else {
		q64, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			fatalln(usage)
		}
		quorum = uint32(q64)

		for i := 1; i < len(args); i += 2 {
			pubkey, err := hex.DecodeString(args[i])
			if err != nil {
				fatalln(usage)
			}
			url := args[i+1]
			signers = append(signers, &config.BlockSigner{
				// Silently truncate the input (which is likely to be an xpub
				// produced by the create-block-keypair subcommand) to
				// bare-pubkey size.
				// TODO(bobg): When the mockhsm can produce bare pubkeys,
				// treat xpubs on input as an error instead.
				Pubkey: pubkey[:ed25519.PublicKeySize],
				Url:    url,
			})
		}
	}

	conf := &config.Config{
		IsGenerator:       true,
		IsSigner:          *isSigner,
		Quorum:            quorum,
		Signers:           signers,
		MaxIssuanceWindow: bc.DurationMillis(*maxIssuanceWindow),
	}

	ctx := context.Background()
	err = config.Configure(ctx, db, rDB, conf)
	if err != nil {
		fatalln("error:", err)
	}

	fmt.Println("blockchain id", conf.BlockchainId)
}

func createBlockKeyPair(db *sql.DB, _ *raft.Service, args []string) {
	if len(args) != 0 {
		fatalln("error: create-block-keypair takes no args")
	}

	hsm := mockhsm.New(db)
	ctx := context.Background()
	pub, err := hsm.Create(ctx, "block_key")
	if err != nil {
		fatalln("error:", err)
	}

	fmt.Printf("%x\n", pub.Pub)
}

func createToken(db *sql.DB, _ *raft.Service, args []string) {
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

	accessTokens := &accesstoken.CredentialStore{DB: db}
	typ := map[bool]string{true: "network", false: "client"}[*flagNet]
	tok, err := accessTokens.Create(context.Background(), args[0], typ)
	if err != nil {
		fatalln("error:", err)
	}
	fmt.Println(tok.Token)
}

func configNongenerator(db *sql.DB, rDB *raft.Service, args []string) {
	const usage = "usage: corectl config [-t token] [-k pubkey] [blockchain-id] [url]"
	var flags flag.FlagSet
	flagT := flags.String("t", "", "generator access `token`")
	flagK := flags.String("k", "", "local `pubkey` for signing blocks")
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

	var blockchainID bc.Hash
	err := blockchainID.UnmarshalText([]byte(args[0]))
	if err != nil {
		fatalln("error: invalid blockchain ID:", err)
	}

	var conf config.Config
	conf.BlockchainId = blockchainID.Proto()
	conf.GeneratorUrl = args[1]
	conf.GeneratorAccessToken = *flagT
	conf.IsSigner = *flagK != ""
	conf.BlockPub = []byte(*flagK)

	ctx := context.Background()
	err = config.Configure(ctx, db, rDB, &conf)
	if err != nil {
		fatalln("error:", err)
	}
}

func fatalln(v ...interface{}) {
	io.Copy(os.Stderr, &logbuf)
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(2)
}

func help(w io.Writer) {
	fmt.Fprintln(w, "usage: corectl [command] [arguments]")
	fmt.Fprint(w, "\nThe commands are:\n\n")
	for name := range commands {
		fmt.Fprintln(w, "\t", name)
	}
	fmt.Fprintln(w)
}

// copied from cmd/cored/main.go
// TODO(tessr): maybe avoid copying this function
func defaultDir() string {
	// TODO(kr): something in ~/Library on darwin?
	return filepath.Join(os.Getenv("HOME"), ".cored")
}
