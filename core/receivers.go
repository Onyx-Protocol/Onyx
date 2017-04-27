package core

import (
	"context"
	"sync"
	"time"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
	chainjson "chain/encoding/json"
	"chain/net/http/reqid"
)

type (
	createAccountPubkeyRequest struct {
		AccountID    string `json:"account_id"`
		AccountAlias string `json:"account_alias"`
	}
	createAccountPubkeyResponse struct {
		Root   chainkd.XPub         `json:"root_xpub"`
		Pubkey ed25519.PublicKey    `json:"pubkey"`
		Path   []chainjson.HexBytes `json:"pubkey_derivation_path"`
	}
)

// POST /create-account-pubkey
func (a *API) createAccountPubkey(ctx context.Context, req createAccountPubkeyRequest) (createAccountPubkeyResponse, error) {
	root, pubkey, path, err := a.accounts.CreatePubkey(ctx, req.AccountID, req.AccountAlias)
	if err != nil {
		return createAccountPubkeyResponse{}, err
	}
	res := createAccountPubkeyResponse{
		Root:   root,
		Pubkey: pubkey,
	}
	for _, p := range path {
		res.Path = append(res.Path, chainjson.HexBytes(p))
	}
	return res, nil
}

// POST /create-account-receiver
func (a *API) createAccountReceiver(ctx context.Context, ins []struct {
	AccountID    string    `json:"account_id"`
	AccountAlias string    `json:"account_alias"`
	ExpiresAt    time.Time `json:"expires_at"`
}) []interface{} {
	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			receiver, err := a.accounts.CreateReceiver(subctx, ins[i].AccountID, ins[i].AccountAlias, ins[i].ExpiresAt)
			if err != nil {
				responses[i] = err
			} else {
				responses[i] = receiver
			}
		}(i)
	}

	wg.Wait()
	return responses
}
