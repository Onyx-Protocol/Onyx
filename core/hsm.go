//+build !prod

package core

import (
	"context"
	"net/http"

	"chain/core/mockhsm"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/net/http/httpjson"
)

func init() {
	errorInfoTab[mockhsm.ErrDuplicateKeyAlias] = errorInfo{400, "CH050", "Alias already exists"}
	errorInfoTab[mockhsm.ErrInvalidAfter] = errorInfo{400, "CH801", "Invalid `after` in query"}
	errorInfoTab[mockhsm.ErrTooManyAliasesToList] = errorInfo{400, "CH802", "Too many aliases to list"}
}

type MockHSMHandler struct {
	MockHSM *mockhsm.HSM
}

func (h *MockHSMHandler) Register(m *http.ServeMux, a *API) {
	needConfig := a.needConfig()

	m.Handle("/mockhsm/create-key", needConfig(h.mockhsmCreateKey))
	m.Handle("/mockhsm/list-keys", needConfig(h.mockhsmListKeys))
	m.Handle("/mockhsm/delkey", needConfig(h.mockhsmDelKey))
	m.Handle("/mockhsm/sign-transaction", needConfig(h.mockhsmSignTemplates))
}

func (h *MockHSMHandler) mockhsmCreateKey(ctx context.Context, in struct{ Alias string }) (result *mockhsm.XPub, err error) {
	return h.MockHSM.XCreate(ctx, in.Alias)
}

func (h *MockHSMHandler) mockhsmListKeys(ctx context.Context, query requestQuery) (page, error) {
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

func (h *MockHSMHandler) mockhsmDelKey(ctx context.Context, xpub chainkd.XPub) error {
	return h.MockHSM.DeleteChainKDKey(ctx, xpub)
}

func (h *MockHSMHandler) mockhsmSignTemplates(ctx context.Context, x struct {
	Txs   []*txbuilder.Template `json:"transactions"`
	XPubs []chainkd.XPub        `json:"xpubs"`
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

func (h *MockHSMHandler) mockhsmSignTemplate(ctx context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte) ([]byte, error) {
	sigBytes, err := h.MockHSM.XSign(ctx, xpub, path, data[:])
	if err == mockhsm.ErrNoKey {
		return nil, nil
	}
	return sigBytes, err
}
