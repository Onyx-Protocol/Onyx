package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"chain/core/rpc"
)

type core struct {
	netTok string
	pubkey string
}

func coreEnv(prefix string) (*rpc.Client, core) {
	var core core
	core.netTok = os.Getenv(prefix + "_NETWORK_TOKEN")
	core.pubkey = os.Getenv(prefix + "_PUBKEY")
	url := os.Getenv(prefix + "_URL")
	clientTok := os.Getenv(prefix + "_CLIENT_TOKEN")

	if url == "" || clientTok == "" || core.netTok == "" || core.pubkey == "" {
		log.Fatalf("please set %s_URL %[1]s_CLIENT_TOKEN %[1]s_NETWORK_TOKEN %[1]s_PUBKEY", prefix)
	}

	client := &rpc.Client{
		BaseURL:     url,
		AccessToken: clientTok,
		Username:    "testnet-resetter", // for user-agent, not auth
		BuildTag:    "none",
	}

	return client, core
}

func main() {
	log.SetFlags(0)
	ctx := context.Background()

	gen, genCore := coreEnv("GENERATOR")
	sig1, sig1Core := coreEnv("SIGNER1")
	sig2, sig2Core := coreEnv("SIGNER2")

	if os.Getenv("AUTH_USER") == "" || os.Getenv("AUTH_TOKEN") == "" {
		log.Fatal("must set heroku user credentials")
	}

	empty := json.RawMessage("{}")
	must(gen.Call(ctx, "/reset", &empty, nil))
	must(sig1.Call(ctx, "/reset", &empty, nil))
	must(sig2.Call(ctx, "/reset", &empty, nil))

	time.Sleep(time.Second) // give them time to restart

	// configure generator
	must(gen.Call(ctx, "/configure", map[string]interface{}{
		"is_signer":    true,
		"block_pub":    genCore.pubkey,
		"is_generator": true,
		"quorum":       2,
		"block_signer_urls": []map[string]interface{}{
			{
				"pubkey":       sig1Core.pubkey,
				"url":          sig1.BaseURL,
				"access_token": sig1Core.netTok,
			},
			{
				"pubkey":       sig2Core.pubkey,
				"url":          sig2.BaseURL,
				"access_token": sig2Core.netTok,
			},
		},
	}, nil))

	time.Sleep(time.Second) // give generator time to restart

	var resp struct {
		BlockchainID string `json:"blockchain_id"`
	}
	must(gen.Call(ctx, "/info", "", &resp))
	log.Println("blockchain ID", resp.BlockchainID)

	// configure signers
	must(sig1.Call(ctx, "/configure", map[string]interface{}{
		"is_signer":              true,
		"block_pub":              sig1Core.pubkey,
		"blockchain_id":          resp.BlockchainID,
		"generator_url":          gen.BaseURL,
		"generator_access_token": genCore.netTok,
	}, nil))
	must(sig2.Call(ctx, "/configure", map[string]interface{}{
		"is_signer":              true,
		"block_pub":              sig2Core.pubkey,
		"blockchain_id":          resp.BlockchainID,
		"generator_url":          gen.BaseURL,
		"generator_access_token": genCore.netTok,
	}, nil))

	method := "PATCH"
	url := "https://api.heroku.com/apps/chain-testnet-info/config-vars"
	r := strings.NewReader(`{"BLOCKCHAIN_ID":"` + resp.BlockchainID + `"}`)
	client := http.Client{}
	req, err := http.NewRequest(method, url, r)
	must(err)
	req.Header.Add("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Add("Content-type", "application/json")
	req.SetBasicAuth(os.Getenv("AUTH_USER"), os.Getenv("AUTH_TOKEN"))
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	must(err)
	os.Stdout.Write(body)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
