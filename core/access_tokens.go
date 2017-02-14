package core

import (
	"context"
	"errors"

	"chain/core/accesstoken"
	"chain/net/http/httpjson"
)

var errCurrentToken = errors.New("token cannot delete itself")

func (a *API) createAccessToken(ctx context.Context, x struct{ ID, Type string }) (*accesstoken.Token, error) {
	return a.AccessTokens.Create(ctx, x.ID, x.Type)
}

func (a *API) listAccessTokens(ctx context.Context, x requestQuery) (*page, error) {
	limit := x.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	tokens, next, err := a.AccessTokens.List(ctx, x.Type, x.After, limit)
	if err != nil {
		return nil, err
	}

	outQuery := x
	outQuery.After = next

	return &page{
		Items:    httpjson.Array(tokens),
		LastPage: len(tokens) < limit,
		Next:     outQuery,
	}, nil
}

func (a *API) deleteAccessToken(ctx context.Context, x struct{ ID string }) error {
	currentID, _, _ := httpjson.Request(ctx).BasicAuth()
	if currentID == x.ID {
		return errCurrentToken
	}
	return a.AccessTokens.Delete(ctx, x.ID)
}
