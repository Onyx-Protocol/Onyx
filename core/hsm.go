package core

import (
	"context"

	"chain/core/mockhsm"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/net/http/httpjson"
)

func (a *API) mockhsmCreateKey(ctx context.Context, in struct{ Alias string }) (result *mockhsm.XPub, err error) {
	return a.MockHSM.XCreate(ctx, in.Alias)
}

func (a *API) mockhsmListKeys(ctx context.Context, query requestQuery) (page, error) {
	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	xpubs, after, err := a.MockHSM.ListKeys(ctx, query.Aliases, query.After, limit)
	if err != nil {
		return page{}, err
	}

	var items []interface{}
	for _, xpub := range xpubs {
		items = append(items, xpub)
	}

	query.After = after

	return page{
		Items:    httpjson.Array(items),
		LastPage: len(xpubs) < limit,
		Next:     query,
	}, nil
}

func (a *API) mockhsmDelKey(ctx context.Context, xpub chainkd.XPub) error {
	return a.MockHSM.DeleteChainKDKey(ctx, xpub)
}

func (a *API) mockhsmSignTemplates(ctx context.Context, x struct {
	Txs   []*txbuilder.Template `json:"transactions"`
	XPubs []chainkd.XPub        `json:"xpubs"`
}) []interface{} {
	resp := make([]interface{}, 0, len(x.Txs))
	for _, tx := range x.Txs {
		err := txbuilder.Sign(ctx, tx, x.XPubs, a.mockhsmSignTemplate)
		if err != nil {
			info, _ := errInfo(err)
			resp = append(resp, info)
		} else {
			resp = append(resp, tx)
		}
	}
	return resp
}

func (a *API) mockhsmSignTemplate(ctx context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte) ([]byte, error) {
	sigBytes, err := a.MockHSM.XSign(ctx, xpub, path, data[:])
	if err == mockhsm.ErrNoKey {
		return nil, nil
	}
	return sigBytes, err
}
