package main

import (
	"context"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"chain/core/pb"
	"chain/core/rpc"
	"chain/env"
	"chain/protocol/bc"
)

var scheduled = env.Bool("SCHEDULED", true)

type core struct {
	netTok string
	pubkey []byte
	url    string
}

func coreEnv(prefix string) (pb.AppClient, core) {
	var (
		c   core
		err error
	)
	c.netTok = os.Getenv(prefix + "_NETWORK_TOKEN")
	c.pubkey, err = hex.DecodeString(os.Getenv(prefix + "_PUBKEY"))
	if err != nil {
		log.Fatalf("bad %s_PUBKEY: %s", prefix, os.Getenv(prefix+"_PUBKEY"))
	}
	c.url = os.Getenv(prefix + "_URL")
	clientTok := os.Getenv(prefix + "_CLIENT_TOKEN")

	if c.url == "" || clientTok == "" || c.netTok == "" || len(c.pubkey) == 0 {
		log.Fatalf("please set %s_URL %[1]s_CLIENT_TOKEN %[1]s_NETWORK_TOKEN %[1]s_PUBKEY", prefix)
	}

	conn, err := rpc.NewGRPCConn(c.url, clientTok, "", "")
	if err != nil {
		log.Fatal("could not create grpc connection")
	}

	client := pb.NewAppClient(conn.Conn)

	return client, c
}

func main() {
	log.SetFlags(0)
	ctx := context.Background()
	env.Parse()

	cur := time.Now()
	max := cur.Add(time.Hour).Weekday()
	min := cur.Add(-1 * time.Hour).Weekday()
	if *scheduled && (min != time.Saturday || max != time.Sunday) {
		log.Println("only run Sunday at midnight +/- an hour")
		os.Exit(0)
	}

	gen, genCore := coreEnv("GENERATOR")
	sig1, sig1Core := coreEnv("SIGNER1")
	sig2, sig2Core := coreEnv("SIGNER2")

	if os.Getenv("HEROKU_API_USER") == "" || os.Getenv("HEROKU_API_KEY") == "" {
		log.Fatal("must set heroku user credentials")
	}

	// scale down testnet bot
	updateHerokuApp("/chain-core-ccte/formation", `{"updates":[{"type":"web", "quantity":0}]}`)

	must(reduce(gen.Reset(ctx, &pb.ResetRequest{})))
	must(reduce(sig1.Reset(ctx, &pb.ResetRequest{})))
	must(reduce(sig2.Reset(ctx, &pb.ResetRequest{})))

	time.Sleep(time.Second) // give them time to restart

	// configure generator
	must(reduce(gen.Configure(ctx, &pb.ConfigureRequest{
		IsSigner:    true,
		BlockPub:    genCore.pubkey,
		IsGenerator: true,
		Quorum:      2,
		BlockSignerUrls: []*pb.ConfigureRequest_BlockSigner{{
			Pubkey:      sig1Core.pubkey,
			Url:         sig1Core.url,
			AccessToken: sig1Core.netTok,
		},
			{
				Pubkey:      sig2Core.pubkey,
				Url:         sig2Core.url,
				AccessToken: sig2Core.netTok,
			}},
	})))

	time.Sleep(time.Second) // give generator time to restart

	resp, err := gen.Info(ctx, nil)
	must(err)

	// configure signers
	must(reduce(sig1.Configure(ctx, &pb.ConfigureRequest{
		IsSigner:             true,
		BlockPub:             sig1Core.pubkey,
		BlockchainId:         resp.BlockchainId,
		GeneratorUrl:         genCore.url,
		GeneratorAccessToken: genCore.netTok,
	})))
	must(reduce(sig2.Configure(ctx, &pb.ConfigureRequest{
		IsSigner:             true,
		BlockPub:             sig2Core.pubkey,
		BlockchainId:         resp.BlockchainId,
		GeneratorUrl:         genCore.url,
		GeneratorAccessToken: genCore.netTok,
	})))

	var blockchainID bc.Hash
	copy(blockchainID[:], resp.BlockchainId)

	// update blockchain id and scale up bot
	updateHerokuApp("/chain-testnet-info/config-vars", `{"BLOCKCHAIN_ID":"`+blockchainID.String()+`"}`)
	updateHerokuApp("/chain-core-ccte/config-vars", `{"BLOCKCHAIN_ID":"`+blockchainID.String()+`"}`)
	updateHerokuApp("/chain-core-ccte/formation", `{"updates":[{"type":"web", "quantity":1}]}`)
}

func updateHerokuApp(endpoint, body string) {
	r := strings.NewReader(body)
	req, err := http.NewRequest("PATCH", "https://api.heroku.com/apps"+endpoint, r)
	must(err)
	req.Header.Add("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Add("Content-type", "application/json")
	req.SetBasicAuth(os.Getenv("HEROKU_API_USER"), os.Getenv("HEROKU_API_KEY"))
	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func reduce(_ interface{}, err error) error {
	return err
}
