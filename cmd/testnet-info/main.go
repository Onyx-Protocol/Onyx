package main

import (
	"encoding/json"
	"log"
	"net/http"

	"chain/env"
)

var (
	addr = env.String("LISTEN", ":8080")
	info = struct {
		GeneratorURL         *string `json:"generator_url"`
		GeneratorAccessToken *string `json:"generator_access_token"`
		BlockchainID         *string `json:"blockchain_id"`
		NetworkRPCVersion    *int    `json:"network_rpc_version"`
		CrosscoreRPCVersion  *int    `json:"crosscore_rpc_version"`
		NextReset            *string `json:"next_reset"`
	}{
		env.String("GENERATOR_URL", ""),
		env.String("GENERATOR_ACCESS_TOKEN", ""),
		env.String("BLOCKCHAIN_ID", ""),
		env.Int("CROSSCORE_RPC_VERSION", 1), // network_rpc_version is a legacy term for crosscore_rpc_version
		env.Int("CROSSCORE_RPC_VERSION", 1),
		env.String("NEXT_RESET", ""),
	}
)

func main() {
	env.Parse()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		json.NewEncoder(w).Encode(info)
	})
	log.Fatal(http.ListenAndServe(*addr, nil))
}
