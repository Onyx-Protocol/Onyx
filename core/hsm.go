package core

import (
	"context"

	"chain/core/mockhsm"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/errors"
	"chain/net/http/httpjson"
)

func (a *api) mockhsmCreateKey(ctx context.Context, in struct{ Alias string }) (result *mockhsm.XPub, err error) {
	result, err = a.hsm.CreateChainKDKey(ctx, in.Alias)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (a *api) mockhsmListKeys(ctx context.Context, query struct{ After string }) (page, error) {
	limit := defGenericPageSize

	xpubs, after, err := a.hsm.ListKeys(ctx, query.After, limit)
	if err != nil {
		return page{}, err
	}

	var items []interface{}
	for _, xpub := range xpubs {
		items = append(items, xpub)
	}

	return page{
		Items:    httpjson.Array(items),
		LastPage: len(xpubs) < limit,
		Next:     requestQuery{After: after},
	}, nil
}

func (a *api) mockhsmDelKey(ctx context.Context, xpub chainkd.XPub) error {
	return a.hsm.DeleteChainKDKey(ctx, xpub)
}

func (a *api) mockhsmSignTemplates(ctx context.Context, x struct {
	Txs   []*txbuilder.Template `json:"transactions"`
	XPubs []string              `json:"xpubs"`
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

func (a *api) mockhsmSignTemplate(ctx context.Context, xpubstr string, path [][]byte, data [32]byte) ([]byte, error) {
	var xpub chainkd.XPub
	err := xpub.UnmarshalText([]byte(xpubstr))
	if err != nil {
		return nil, errors.Wrap(err, "parsing xpub")
	}
	sigBytes, err := a.hsm.SignWithChainKDKey(ctx, xpub, path, data[:])
	if err == mockhsm.ErrNoKey {
		return nil, nil
	}
	return sigBytes, err
}
