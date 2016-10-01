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
	"chain/core/mockhsm"
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
	"config":               {configNongenerator},
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
	const usage = "error: corectl config-generator [<quorum> <<pubkey> <url>...>]"
	var (
		isSigner bool
		quorum   int
		signers  []core.ConfigSigner
		err      error
	)
	if len(args) == 0 {
		isSigner = true
		quorum = 1
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
				Pubkey: pubkey,
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

func configNongenerator(db *sql.DB, args []string) {
	if len(args) != 2 && len(args) != 3 {
		fatalln("error: corectl config <blockchain-id> <generator-url> [block-pubkey]")
	}

	var config core.Config
	err := config.BlockchainID.UnmarshalText([]byte(args[0]))
	if err != nil {
		fatalln("error: invalid blockchain ID:", err)
	}
	config.GeneratorURL = args[1]
	if len(args) > 2 {
		config.IsSigner = true
		config.BlockXPub = args[2]
	}

	ctx := context.Background()
	err = core.Configure(ctx, db, &config)
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
