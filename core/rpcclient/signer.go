package rpcclient

import (
	"encoding/json"

	"golang.org/x/net/context"

	"chain/net/rpc"
)

// GetSignatureForSerializedBlock sends a sign-block RPC request to
// the given signer, with the block already serialized for transport.
func GetSignatureForSerializedBlock(ctx context.Context, signerURL string, block []byte) ([]byte, error) {
	var result []byte
	err := rpc.Call(ctx, signerURL, "/rpc/signer/sign-block", (*json.RawMessage)(&block), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
