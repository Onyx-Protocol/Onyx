package core

import (
	"errors"

	"golang.org/x/net/context"

	"chain/core/pb"
	"chain/net/http/httpjson"
)

var errCurrentToken = errors.New("token cannot delete itself")

func (h *Handler) CreateAccessToken(ctx context.Context, in *pb.CreateAccessTokenRequest) (*pb.CreateAccessTokenResponse, error) {
	resp, err := h.AccessTokens.Create(ctx, in.Id, in.Type)
	if err != nil {
		return nil, err
	}
	return &pb.CreateAccessTokenResponse{
		Token: &pb.AccessToken{
			Id:        resp.ID,
			Token:     resp.Token,
			Type:      resp.Type,
			CreatedAt: resp.Created.String(),
		},
	}, nil
}

func (h *Handler) ListAccessTokens(ctx context.Context, in *pb.ListAccessTokensQuery) (*pb.ListAccessTokensResponse, error) {
	limit := int(in.PageSize)
	if limit == 0 {
		limit = defGenericPageSize
	}

	tokens, next, err := h.AccessTokens.List(ctx, in.Type, in.After, limit)
	if err != nil {
		return nil, err
	}

	pbTokens := make([]*pb.AccessToken, len(tokens))
	for i, t := range tokens {
		pbTokens[i] = &pb.AccessToken{
			Id:        t.ID,
			Type:      t.Type,
			CreatedAt: t.Created.String(),
		}
	}

	outQuery := in
	outQuery.After = next

	return &pb.ListAccessTokensResponse{
		Items:    pbTokens,
		LastPage: len(tokens) < limit,
		Next:     outQuery,
	}, nil
}

func (h *Handler) DeleteAccessToken(ctx context.Context, in *pb.DeleteAccessTokenRequest) (*pb.ErrorResponse, error) {
	currentID, _, _ := httpjson.Request(ctx).BasicAuth()
	if currentID == in.Id {
		return nil, errCurrentToken
	}
	return nil, h.AccessTokens.Delete(ctx, in.Id)
}
