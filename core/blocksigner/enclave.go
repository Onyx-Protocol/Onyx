package blocksigner

import (
	"context"
	"fmt"
	"strings"

	"chain/core/rpc"
	"chain/crypto/ed25519"
	"chain/encoding/json"
	"chain/log"
	"chain/protocol/bc/legacy"
)

// EnclaveClient implements the Signer interface by calling
// Chain Enclave to sign blocks.
type EnclaveClient struct {
	// URLs is called on every Sign call to retrieve the URLs
	// and access tokens for Chain Enclave.
	URLs       func() [][]string
	BaseClient rpc.Client
}

type signRequestBody struct {
	Block *legacy.BlockHeader `json:"block"`
	Pub   json.HexBytes       `json:"pubkey"`
}

func (ec EnclaveClient) Sign(ctx context.Context, pk ed25519.PublicKey, bh *legacy.BlockHeader) ([]byte, error) {
	body := signRequestBody{Block: bh, Pub: json.HexBytes(pk[:])}

	// grab the latest set of hsms from the configuration
	hsmURLs := ec.URLs()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan result, len(hsmURLs))
	for _, tup := range hsmURLs {
		// make a copy of the base rpc client on the stack so we
		// can modify it with the URL and access token
		client := ec.BaseClient
		client.BaseURL = tup[0]
		client.AccessToken = tup[1]

		// If the provided access token is just a password with no
		// user, use a default basic auth username of 'chaincore'.
		if !strings.Contains(client.AccessToken, ":") {
			client.AccessToken = "chaincore:" + client.AccessToken
		}

		go func() {
			var signature []byte
			err := client.Call(ctx, "/sign-block", body, &signature)
			ch <- result{signature: signature, err: err}
		}()
	}

	var err error
	for i := 0; i < len(hsmURLs); i++ {
		res := <-ch
		if res.err == nil {
			return res.signature, nil
		}
		err = res.err
		log.Error(ctx, err, fmt.Sprintf("Unable to sign block at height %d with enclave %s", bh.Height, hsmURLs[i][0]))
	}
	return nil, err
}

type result struct {
	err       error
	signature []byte
}
