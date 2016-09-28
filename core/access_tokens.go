package core

import (
	"context"

	"chain/core/accesstoken"
)

func createAccessToken(ctx context.Context, x struct{ ID, Type string }) (*accesstoken.Token, error) {
	return accesstoken.Create(ctx, x.ID, x.Type)
}

func listAccessTokens(ctx context.Context, x requestQuery) (*page, error) {
	limit := x.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	tokens, next, err := accesstoken.List(ctx, x.After, limit)
	if err != nil {
		return nil, err
	}

	outQuery := x
	x.After = next

	return &page{
		Items:    tokens,
		LastPage: len(tokens) < limit,
		Next:     outQuery,
	}, nil
}

func deleteAccessToken(ctx context.Context, x struct{ ID string }) error {
	return accesstoken.Delete(ctx, x.ID)
}
