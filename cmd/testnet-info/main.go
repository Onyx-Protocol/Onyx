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
		ProposerURL       *string `json:"proposer_url"`
		BlockchainID      *string `json:"blockchain_id"`
		NetworkRPCVersion *int    `json:"network_rpc_version"`
		NextReset         *string `json:"next_reset"`
	}{
		env.String("PROPOSER_URL", ""),
		env.String("BLOCKCHAIN_ID", ""),
		env.Int("NETWORK_RPC_VERSION", 1),
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
