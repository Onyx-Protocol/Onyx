package rpcclient

import (
	"encoding/json"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/crypto"
	"chain/net/rpc"
)

// GetSignatureForSerializedBlock sends a sign-block RPC request to
// the given signer, with the block already serialized for transport.
func GetSignatureForSerializedBlock(ctx context.Context, signerURL string, block []byte) (*btcec.Signature, error) {
	var result btcec.Signature
	err := rpc.Call(ctx, signerURL, "/rpc/signer/sign-block", json.RawMessage(block), (*crypto.Signature)(&result))
	if err != nil {
		return nil, err
	}
	return &result, nil
}
