package blocksigner

import (
	"context"
	"time"

	"chain/core/rpc"
	"chain/crypto/ed25519"
	"chain/encoding/json"
	"chain/protocol/bc/legacy"
)

// enclaveTimeout is the time to wait for an HSM's response to a
// sign-block request before moving on to the next HSM. Because
// blocks are expected to be generated every second, one second
// retireving a signature would be a very high latency.
const enclaveTimeout = time.Second

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

	// make a copy of the base rpc client on the stack so we
	// can modify it with the HSM URLs and access tokens
	client := ec.BaseClient

	// grab the latest set of hsms from the configuration
	hsmURLs := ec.URLs()

	var signature []byte
	var err error
	for _, tup := range hsmURLs {
		client.BaseURL = tup[0]
		client.AccessToken = tup[1]

		callCtx, cancel := context.WithTimeout(ctx, enclaveTimeout)
		err = client.Call(callCtx, "/sign-block", body, &signature)
		cancel()
		if err == nil {
			return signature, nil
		}
	}
	return nil, err
}
