//+build !no_mockhsm

package core

import (
	"context"

	"chain/core/mockhsm"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/net/http/httperror"
	"chain/net/http/httpjson"
)

func init() {
	errorFormatter.Errors[mockhsm.ErrDuplicateKeyAlias] = httperror.Info{400, "CH050", "Alias already exists"}
	errorFormatter.Errors[mockhsm.ErrInvalidAfter] = httperror.Info{400, "CH801", "Invalid `after` in query"}
	errorFormatter.Errors[mockhsm.ErrTooManyAliasesToList] = httperror.Info{400, "CH802", "Too many aliases to list"}
}

// MockHSM configures the Core to expose the MockHSM endpoints. It
// is only included in non-production builds.
func MockHSM(hsm *mockhsm.HSM) RunOption {
	return func(a *API) {
		h := &mockHSMHandler{MockHSM: hsm}

		needConfig := a.needConfig()
		a.mux.Handle("/mockhsm/create-block-key", jsonHandler(h.mockhsmCreateBlockKey))
		a.mux.Handle("/mockhsm/create-key", needConfig(h.mockhsmCreateKey))
		a.mux.Handle("/mockhsm/list-keys", needConfig(h.mockhsmListKeys))
		a.mux.Handle("/mockhsm/delkey", needConfig(h.mockhsmDelKey))
		a.mux.Handle("/mockhsm/sign-transaction", needConfig(h.mockhsmSignTemplates))
	}
}

type mockHSMHandler struct {
	MockHSM *mockhsm.HSM
}

func (h *mockHSMHandler) mockhsmCreateBlockKey(ctx context.Context) (result *mockhsm.Pub, err error) {
	return h.MockHSM.Create(ctx, "block_key")
}

func (h *mockHSMHandler) mockhsmCreateKey(ctx context.Context, in struct{ Alias string }) (result *mockhsm.XPub, err error) {
	return h.MockHSM.XCreate(ctx, in.Alias)
}

func (h *mockHSMHandler) mockhsmListKeys(ctx context.Context, query requestQuery) (page, error) {
	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	xpubs, after, err := h.MockHSM.ListKeys(ctx, query.Aliases, query.After, limit)
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

func (h *mockHSMHandler) mockhsmDelKey(ctx context.Context, xpub chainkd.XPub) error {
	return h.MockHSM.DeleteChainKDKey(ctx, xpub)
}

func (h *mockHSMHandler) mockhsmSignTemplates(ctx context.Context, x struct {
	Txs   []*txbuilder.Template `json:"transactions"`
	XPubs []chainkd.XPub        `json:"xpubs"`
}) []interface{} {
	resp := make([]interface{}, 0, len(x.Txs))
	for _, tx := range x.Txs {
		err := txbuilder.Sign(ctx, tx, x.XPubs, h.mockhsmSignTemplate)
		if err != nil {
			info := errorFormatter.Format(err)
			resp = append(resp, info)
		} else {
			resp = append(resp, tx)
		}
	}
	return resp
}

func (h *mockHSMHandler) mockhsmSignTemplate(ctx context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte) ([]byte, error) {
	sigBytes, err := h.MockHSM.XSign(ctx, xpub, path, data[:])
	if err == mockhsm.ErrNoKey {
		return nil, nil
	}
	return sigBytes, err
}
