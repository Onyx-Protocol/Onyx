package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"chain/core/generator"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
	"chain/log"
	"chain/protocol"
)

// config vars
var (
	dbURL = env.String("DATABASE_URL", "postgres:///core?sslmode=disable")
)

// We collect log output in this buffer,
// and display it only when there's an error.
var logbuf bytes.Buffer

type command struct {
	f         func(*sql.DB, []string)
	shortHelp string
}

var commands = map[string]*command{
	"init": {initblock, "init [quorum] [key...]"},
}

func main() {
	log.SetOutput(&logbuf)
	env.Parse()
	sql.Register("schemadb", pg.SchemaDriver("corectl"))
	db, err := sql.Open("schemadb", *dbURL)
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

func initblock(db *sql.DB, args []string) {
	if len(args) == 0 {
		fatalln("error: please provide a quorum size")
	}
	quorum, err := strconv.Atoi(args[0])
	args = args[1:]
	if err != nil {
		fatalln("error:", err)
	}
	if quorum > len(args) {
		fatalln("error: quorum size requires more keys than provided")
	}

	var keys []ed25519.PublicKey
	for _, s := range args {
		b, err := hex.DecodeString(s)
		if err != nil {
			fatalln("error:", err)
		}
		keys = append(keys, b)
	}

	block, err := protocol.NewGenesisBlock(keys, quorum, time.Now())
	if err != nil {
		fatalln("error:", err)
	}

	ctx := context.Background()
	err = generator.SaveInitialBlock(ctx, db, block)
	if err != nil {
		fatalln("error:", err)
	}

	fmt.Printf("block created: %+v\n\n", block)
	fmt.Println("initial block hash", block.Hash())
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
