package core

import (
	"context"

	"chain/core/mockhsm"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/hd25519"
	"chain/encoding/json"
	"chain/errors"
	"chain/net/http/httpjson"
)

func (a *api) mockhsmCreateKey(ctx context.Context, in struct{ Alias *string }) (result *mockhsm.XPub, err error) {
	result, err = a.hsm.CreateKey(ctx, in.Alias)
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

func (a *api) mockhsmDelKey(ctx context.Context, xpubBytes json.HexBytes) error {
	xpub, err := hd25519.XPubFromBytes(xpubBytes)
	if err != nil {
		return err
	}
	return a.hsm.DelKey(ctx, xpub)
}

func (a *api) mockhsmSignTemplates(ctx context.Context, tpls []*txbuilder.Template) []interface{} {
	resp := make([]interface{}, 0, len(tpls))
	for _, tpl := range tpls {
		err := txbuilder.Sign(ctx, tpl, a.mockhsmSignTemplate)
		if err != nil {
			info, _ := errInfo(err)
			resp = append(resp, info)
		} else {
			resp = append(resp, tpl)
		}
	}
	return resp
}

func (a *api) mockhsmSignTemplate(ctx context.Context, xpubstr string, path []uint32, data [32]byte) ([]byte, error) {
	xpub, err := hd25519.XPubFromString(xpubstr)
	if err != nil {
		return nil, errors.Wrap(err, "parsing xpub")
	}
	sigBytes, err := a.hsm.Sign(ctx, xpub, path, data[:])
	if err == mockhsm.ErrNoKey {
		return nil, nil
	}
	return sigBytes, err
}
