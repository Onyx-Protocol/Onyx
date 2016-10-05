package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"

	"chain/core"
	"chain/core/accesstoken"
	"chain/core/mockhsm"
	"chain/crypto/ed25519"
	"chain/database/sql"
	"chain/env"
	"chain/log"
)

// config vars
var (
	dbURL = env.String("DATABASE_URL", "postgres:///core?sslmode=disable")
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

func configGenerator(db *sql.DB, args []string) {
	const usage = "error: corectl config-generator [-s] [<quorum> <<pubkey> <url>...>]"
	var (
		isSigner bool
		quorum   int
		signers  []core.ConfigSigner
		err      error
	)

	if len(args) > 0 && args[0] == "-s" {
		isSigner = true
		args = args[1:]
	}

	if len(args) == 0 {
		if isSigner {
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
			url := args[i+1]
			signers = append(signers, core.ConfigSigner{
				// Silently truncate the input (which is likely to be an xpub
				// produced by the create-block-keypair subcommand) to
				// bare-pubkey size.
				// TODO(bobg): When the mockhsm can produce bare pubkeys,
				// treat xpubs on input as an error instead.
				Pubkey: pubkey[:ed25519.PublicKeySize],
				URL:    url,
			})
		}
	}

	config := &core.Config{
		IsGenerator: true,
		IsSigner:    isSigner,
		Quorum:      quorum,
		Signers:     signers,
	}

	err = initSchema(db)
	if err != nil {
		fatalln("error: init schema", err)
	}
	ctx := context.Background()
	err = core.Configure(ctx, db, config)
	if err != nil {
		fatalln("error:", err)
	}

	fmt.Println("blockchain id", config.BlockchainID)
}

func createBlockKeyPair(db *sql.DB, args []string) {
	if len(args) != 0 {
		fatalln("error: create-block-keypair takes no args")
	}

	hsm := mockhsm.New(db)
	ctx := context.Background()
	xpub, err := hsm.CreateKey(ctx, "block_key")
	if err != nil {
		fatalln("error:", err)
	}

	fmt.Println("block xpub:", xpub.XPub.String())
}

func createToken(db *sql.DB, args []string) {
	var id, typ string
	if len(args) == 1 {
		id, typ = args[0], "client"
	} else if len(args) == 2 && args[0] == "-net" {
		id, typ = args[1], "network"
	} else {
		fatalln("usage: corectl create-token [-net] [id]")
	}

	tok, err := accesstoken.Create(context.Background(), id, typ)
	if err != nil {
		fatalln("error:", err)
	}
	fmt.Printf("%s:%s\n", tok.ID, tok.Token)
}

func configNongenerator(db *sql.DB, args []string) {
	errUsage := "error: corectl config <blockchain-id> <generator-url> [-t <generator-access-token>] [-k <block-pubkey>]"
	if len(args) < 2 {
		fatalln(errUsage)
	}

	var config core.Config
	err := config.BlockchainID.UnmarshalText([]byte(args[0]))
	if err != nil {
		fatalln("error: invalid blockchain ID:", err)
	}
	config.GeneratorURL = args[1]

	for args = args[2:]; len(args) > 0; args = args[2:] {
		if len(args) < 2 {
			fatalln(errUsage)
		}

		switch args[0] {
		case "-t":
			config.GeneratorAccessToken = args[1]
		case "-k":
			config.IsSigner = true
			config.BlockXPub = args[1]
		default:
			fatalln(errUsage)
		}
	}

	err = initSchema(db)
	if err != nil {
		fatalln("error: init schema", err)
	}
	ctx := context.Background()
	err = core.Configure(ctx, db, &config)
	if err != nil {
		fatalln("error:", err)
	}
}

func reset(db *sql.DB, args []string) {
	if len(args) != 0 {
		fatalln("error: reset takes no args")
	}

	ctx := context.Background()
	err := core.Reset(ctx, db)
	if err != nil {
		fatalln("error:", err)
	}
}

func initSchema(db *sql.DB) error {
	ctx := context.Background()
	const q = `
		SELECT count(*) FROM pg_tables
		WHERE schemaname='public' AND tablename='migrations'
	`
	var n int
	err := db.QueryRow(ctx, q).Scan(&n)
	if err != nil {
		return err
	} else if n > 0 {
		return nil // already initialized
	}
	_, err = db.Exec(ctx, core.Schema())
	return err
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
