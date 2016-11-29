package core

import (
	"context"

	"chain-stealth/core/mockhsm"
	"chain-stealth/core/txbuilder"
	"chain-stealth/crypto/ed25519/chainkd"
	"chain-stealth/errors"
	"chain-stealth/net/http/httpjson"
)

func (h *Handler) mockhsmCreateKey(ctx context.Context, in struct{ Alias string }) (result *mockhsm.XPub, err error) {
	result, err = h.HSM.XCreate(ctx, in.Alias)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (h *Handler) mockhsmListKeys(ctx context.Context, query requestQuery) (page, error) {
	limit := defGenericPageSize

	xpubs, after, err := h.HSM.ListKeys(ctx, query.Aliases, query.After, limit)
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

func (h *Handler) mockhsmDelKey(ctx context.Context, xpub chainkd.XPub) error {
	return h.HSM.DeleteChainKDKey(ctx, xpub)
}

func (h *Handler) mockhsmSignTemplates(ctx context.Context, x struct {
	Txs   []*txbuilder.Template `json:"transactions"`
	XPubs []string              `json:"xpubs"`
}) []interface{} {
	resp := make([]interface{}, 0, len(x.Txs))
	for _, tx := range x.Txs {
		err := txbuilder.Sign(ctx, tx, x.XPubs, h.mockhsmSignTemplate)
		if err != nil {
			info, _ := errInfo(err)
			resp = append(resp, info)
		} else {
			resp = append(resp, tx)
		}
	}
	return resp
}

func (h *Handler) mockhsmSignTemplate(ctx context.Context, xpubstr string, path [][]byte, data [32]byte) ([]byte, error) {
	var xpub chainkd.XPub
	err := xpub.UnmarshalText([]byte(xpubstr))
	if err != nil {
		return nil, errors.Wrap(err, "parsing xpub")
	}
	sigBytes, err := h.HSM.XSign(ctx, xpub, path, data[:])
	if err == mockhsm.ErrNoKey {
		return nil, nil
	}
	return sigBytes, err
}
