package core

import (
	"context"
	stdjson "encoding/json"
	"sync"

	"chain/encoding/json"
	"chain/errors"
	"chain/net/http/httpjson"
)

// POST /create-control-program
func (h *Handler) createControlProgram(ctx context.Context, ins []struct {
	Type   string
	Params stdjson.RawMessage
}) interface{} {

	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			defer wg.Done()
			var (
				prog interface{}
				err  error
			)
			switch ins[i].Type {
			case "account":
				prog, err = h.createAccountControlProgram(ctx, ins[i].Params)
			default:
				err = errors.WithDetailf(httpjson.ErrBadRequest, "unknown control program type %q", ins[i].Type)
			}
			if err != nil {
				logHTTPError(ctx, err)
				responses[i], _ = errInfo(err)
			} else {
				responses[i] = prog
			}
		}(i)
	}

	wg.Wait()
	return responses
}

func (h *Handler) createAccountControlProgram(ctx context.Context, input []byte) (interface{}, error) {
	var parsed struct {
		AccountAlias string `json:"account_alias"`
		AccountID    string `json:"account_id"`
	}
	err := stdjson.Unmarshal(input, &parsed)
	if err != nil {
		return nil, errors.WithDetailf(httpjson.ErrBadRequest, "bad parameters for account control program")
	}

	accountID := parsed.AccountID
	if accountID == "" {
		acc, err := h.Accounts.FindByAlias(ctx, parsed.AccountAlias)
		if err != nil {
			return nil, err
		}
		accountID = acc.ID
	}

	controlProgram, err := h.Accounts.CreateControlProgram(ctx, accountID, false)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"control_program": json.HexBytes(controlProgram),
	}
	return ret, nil
}
