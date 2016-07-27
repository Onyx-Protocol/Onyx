package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/txdb"
	"chain/cos"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
	"chain/log"
)

// config vars
var (
	dbURL    = env.String("DB_URL", "postgres:///core?sslmode=disable")
	blockKey = env.String("BLOCK_KEY", "2c1f68880327212b6aa71d7c8e0a9375451143352d5c760dc38559f1159c84ce")
)

// We collect log output in this buffer,
// and display it only when there's an error.
var logbuf bytes.Buffer

type command struct {
	f         func(*sql.DB, []string)
	shortHelp string
}

var commands = map[string]*command{
	"adduser":    {addUser, "adduser [email] [password] [role]"},
	"addproject": {addProject, "addproject [name]"},
	"genesis":    {genesis, "genesis"},
	"boot":       {boot, "boot [email] [password]"},
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
	if len(os.Args)-1 != len(strings.Fields(cmd.shortHelp)) {
		fmt.Fprintln(os.Stderr, "usage: corectl", cmd.shortHelp)
		os.Exit(1)
	}
	cmd.f(db, os.Args[2:])
}

func addUser(db *sql.DB, args []string) {
	ctx := pg.NewContext(context.Background(), db)
	u, err := appdb.CreateUser(ctx, args[0], args[1], args[2])
	if err != nil {
		fatalln("error:", err)
	}
	fmt.Printf("user created: %+v\n", *u)
}

func addProject(db *sql.DB, args []string) {
	ctx := pg.NewContext(context.Background(), db)
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		fatalln("begin")
	}
	defer dbtx.Rollback(ctx)

	proj, err := appdb.CreateProject(ctx, args[0])
	if err != nil {
		fatalln("error:", err)
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		fatalln("error:", err)
	}

	fmt.Printf("project created: %+v\n", *proj)
}

func genesis(db *sql.DB, args []string) {
	keyBytes, err := hex.DecodeString(*blockKey)
	if err != nil {
		fatalln("error:", err)
	}

	privKey, err := hd25519.PrvFromBytes(keyBytes)
	if err != nil {
		fatalln("error:", err)
	}
	pubKey := privKey.Public().(ed25519.PublicKey)

	ctx := pg.NewContext(context.Background(), db)

	store, pool := txdb.New(db)
	fc, err := cos.NewFC(ctx, store, pool, nil, nil)
	if err != nil {
		fatalln("error:", err)
	}

	b, err := fc.UpsertGenesisBlock(ctx, []ed25519.PublicKey{pubKey}, 1, time.Now())
	if err != nil {
		fatalln("error:", err)
	}
	fmt.Printf("block created: %+v\n", b)
}

func boot(db *sql.DB, args []string) {
	ctx := pg.NewContext(context.Background(), db)
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		fatalln("begin")
	}
	defer dbtx.Rollback(ctx)

	u, err := appdb.CreateUser(ctx, args[0], args[1], "admin")
	if err != nil {
		fatalln(err)
	}

	tok, err := appdb.CreateAuthToken(ctx, u.ID, "api", nil)
	if err != nil {
		fatalln(err)
	}

	proj, err := appdb.CreateProject(ctx, "proj")
	if err != nil {
		fatalln(err)
	}

	mpub, mpriv := genKey()
	mn, err := appdb.InsertManagerNode(ctx, proj.ID, "manager", mpub, mpriv, 0, 1, nil)
	if err != nil {
		fatalln(err)
	}

	ipub, ipriv := genKey()
	in, err := appdb.InsertIssuerNode(ctx, proj.ID, "issuer", ipub, ipriv, 1, nil)
	if err != nil {
		fatalln(err)
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		fatalln(err)
	}

	result, _ := json.MarshalIndent(map[string]string{
		"userID":        u.ID,
		"tokenID":       tok.ID,
		"tokenSecret":   tok.Secret,
		"projectID":     proj.ID,
		"managerXPRV":   mpriv[0].String(),
		"managerNodeID": mn.ID,
		"issuerXPRV":    ipriv[0].String(),
		"issuerNodeID":  in.ID,
	}, "", "  ")
	fmt.Printf("%s\n", result)
}

func genKey() (pub []*hd25519.XPub, priv []*hd25519.XPrv) {
	xprv, xpub, err := hd25519.NewXKeys(nil)
	if err != nil {
		fatalln(err)
	}
	pub = append(pub, xpub)
	priv = append(priv, xprv)
	return
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
