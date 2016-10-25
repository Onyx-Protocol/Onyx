package core

import (
	"context"
	"errors"

	"chain/core/accesstoken"
	"chain/net/http/httpjson"
)

var errCurrentToken = errors.New("token cannot delete itself")

func (h *Handler) createAccessToken(ctx context.Context, x struct{ ID, Type string }) (*accesstoken.Token, error) {
	return h.AccessTokens.Create(ctx, x.ID, x.Type)
}

func (h *Handler) listAccessTokens(ctx context.Context, x requestQuery) (*page, error) {
	limit := x.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	tokens, next, err := h.AccessTokens.List(ctx, x.Type, x.After, limit)
	if err != nil {
		return nil, err
	}

	outQuery := x
	x.After = next

	return &page{
		Items:    httpjson.Array(tokens),
		LastPage: len(tokens) < limit,
		Next:     outQuery,
	}, nil
}

func (h *Handler) deleteAccessToken(ctx context.Context, x struct{ ID string }) error {
	currentID, _, _ := httpjson.Request(ctx).BasicAuth()
	if currentID == x.ID {
		return errCurrentToken
	}
	return h.AccessTokens.Delete(ctx, x.ID)
}
