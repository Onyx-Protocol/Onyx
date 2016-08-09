package core

import (
	"encoding/json"

	"golang.org/x/net/context"

	"chain/core/account"
	"chain/errors"
	"chain/net/http/httpjson"
)

// POST /createControlProgram
func createControlProgram(ctx context.Context, in struct {
	Type       string
	Parameters json.RawMessage
}) (interface{}, error) {
	if in.Type == "account" {
		return createAccountControlProgram(ctx, in.Parameters)
	}

	return nil, errors.WithDetailf(httpjson.ErrBadRequest, "unknown control program type %q", in.Type)
}

func createAccountControlProgram(ctx context.Context, input []byte) (interface{}, error) {
	var parsed struct {
		AccountId string `json:"account_id"`
	}
	err := json.Unmarshal(input, &parsed)
	if err != nil {
		return nil, errors.WithDetailf(httpjson.ErrBadRequest, "no 'account_id' parameter sent")
	}

	controlProgram, err := account.CreateControlProgram(ctx, parsed.AccountId)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"program": controlProgram,
	}
	return ret, nil
}
