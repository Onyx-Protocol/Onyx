package core

import (
	"context"
	"encoding/json"

	"chain/core/accesstoken"
	"chain/errors"
	"chain/log"
	"chain/net/http/authz"
	"chain/net/http/httpjson"
)

var errCurrentToken = errors.New("token cannot delete itself")

func (a *API) createAccessToken(ctx context.Context, x struct{ ID, Type string }) (*accesstoken.Token, error) {
	token, err := a.accessTokens.Create(ctx, x.ID, x.Type)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if x.Type == "" {
		return token, nil
	}

	data := map[string]interface{}{
		"id": token.ID,
	}
	guardData, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	var grant *authz.Grant

	// Type is deprecated; however, for backward compatibility, using the
	// Type field will create a grant associated with this new token.
	switch x.Type {
	case "client":
		grant = &authz.Grant{
			GuardType: "access_token",
			GuardData: guardData,
			Policy:    "client-readwrite",
		}
	case "network":
		grant = &authz.Grant{
			GuardType: "access_token",
			GuardData: guardData,
			Policy:    "crosscore",
		}
	default:
		// We've already returned if x.Type wasn't specified, so this must be a bad type.
		return nil, accesstoken.ErrBadType
	}
	err = a.sdb.Exec(ctx, a.grants.Save(ctx, grant))
	if err != nil {
		return nil, errors.Wrap(err)
	}

	token.Type = x.Type // deprecated

	return token, nil
}

func (a *API) listAccessTokens(ctx context.Context, x requestQuery) (*page, error) {
	limit := x.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	tokens, next, err := a.accessTokens.List(ctx, x.Type, x.After, limit)
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
	err := a.accessTokens.Delete(ctx, x.ID)
	if err != nil {
		return err
	}

	err = a.sdb.Exec(ctx, a.deleteGrantsByAccessToken(ctx, x.ID))
	if err != nil {
		// well, technically we did delete the access token, so don't return the error
		// TODO(tessr): make this whole operation atomic, such that we either delete
		// both the access token and its grants, or we return a failure.
		log.Printkv(ctx, log.KeyError, err, "at", "revoking grants for access token", "token", x.ID)
	}
	return nil
}
