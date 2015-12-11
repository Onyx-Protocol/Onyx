// bootdb bootstraps the database to a minimal functional state
//
//   user
//   auth token
//   project (with membership)
//   admin node
//   manager node (with keys)
//   issuer node (with keys)
//   genesis block
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/btcsuite/btcutil/hdkeychain"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/database/pg"
	"chain/env"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/log"
)

// config vars
var dbURL = env.String("DB_URL", "postgres:///api?sslmode=disable")

var (
	db     *sql.DB
	logbuf bytes.Buffer
)

func main() {
	env.Parse()
	log.SetOutput(&logbuf)

	if len(os.Args) != 3 {
		fatal("usage: bootdb email password")
	}

	sql.Register("schemadb", pg.SchemaDriver("bootdb"))
	db, err := sql.Open("schemadb", *dbURL)
	if err != nil {
		fatal(err)
	}

	appdb.Init(db)
	ctx := pg.NewContext(context.Background(), db)
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		fatal("begin")
	}
	defer dbtx.Rollback()

	u, err := appdb.CreateUser(ctx, os.Args[1], os.Args[2])
	if err != nil {
		fatal(err)
	}

	tok, err := appdb.CreateAuthToken(ctx, u.ID, "api", nil)
	if err != nil {
		fatal(err)
	}

	proj, err := appdb.CreateProject(ctx, "proj", u.ID)
	if err != nil {
		fatal(err)
	}

	adminNode, err := appdb.InsertAdminNode(ctx, proj.ID, "admin")
	if err != nil {
		fatal(err)
	}

	mpub, mpriv := genKey()
	mn, err := appdb.InsertManagerNode(ctx, proj.ID, "manager", mpub, mpriv)
	if err != nil {
		fatal(err)
	}

	ipub, ipriv := genKey()
	in, err := appdb.InsertIssuerNode(ctx, proj.ID, "issuer", ipub, ipriv)
	if err != nil {
		fatal(err)
	}

	block := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:   bc.NewBlockVersion,
			Timestamp: uint64(time.Now().Unix()),
		},
	}
	const q = `
		INSERT INTO blocks (block_hash, height, data)
		VALUES ($1, $2, $3)
	`
	_, err = pg.FromContext(ctx).Exec(q, block.Hash(), block.Height, block)
	if err != nil {
		fatal(err)
	}

	err = dbtx.Commit()
	if err != nil {
		fatal(err)
	}

	result, _ := json.MarshalIndent(map[string]string{
		"userID":           u.ID,
		"tokenID":          tok.ID,
		"tokenSecret":      tok.Secret,
		"projectID":        proj.ID,
		"adminNodeID":      adminNode.ID,
		"managerXPRV":      mpriv[0].String(),
		"managerNodeID":    mn.ID,
		"issuerXPRV":       ipriv[0].String(),
		"issuerNodeID":     in.ID,
		"genesisBlockHash": block.Hash().String(),
	}, "", "  ")
	fmt.Printf("%s\n", result)
}

func genKey() (pub, priv []*hdkey.XKey) {
	pk, sk, err := newKey()
	if err != nil {
		fatal(err)
	}
	pub = append(pub, pk)
	priv = append(priv, sk)
	return
}

func newKey() (pub, priv *hdkey.XKey, err error) {
	seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating key seed")
	}
	xprv, err := hdkeychain.NewMaster(seed)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating root xprv")
	}
	xpub, err := xprv.Neuter()
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting root xpub")
	}
	return &hdkey.XKey{ExtendedKey: *xpub}, &hdkey.XKey{ExtendedKey: *xprv}, nil
}

func fatal(v interface{}) {
	io.Copy(os.Stderr, &logbuf)
	panic(v)
}
