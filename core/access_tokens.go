package core

import (
	"context"
	"errors"

	"chain/core/accesstoken"
	"chain/net/http/authn"
	"chain/net/http/httpjson"
)

var errCurrentToken = errors.New("token cannot delete itself")

func createAccessToken(ctx context.Context, x struct{ ID, Type string }) (*accesstoken.Token, error) {
	return accesstoken.Create(ctx, x.ID, x.Type)
}

func listAccessTokens(ctx context.Context, x requestQuery) (*page, error) {
	limit := x.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	tokens, next, err := accesstoken.List(ctx, x.Type, x.After, limit)
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

func deleteAccessToken(ctx context.Context, x struct{ ID string }) error {
	currentID := authn.UsernameFromContext(ctx)
	if currentID == x.ID {
		return errCurrentToken
	}
	return accesstoken.Delete(ctx, x.ID)
}
