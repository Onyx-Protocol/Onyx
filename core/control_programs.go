package core

import (
	"context"
	stdjson "encoding/json"
	"sync"
	"time"

	"chain/encoding/json"
	"chain/errors"
	"chain/net/http/httpjson"
	"chain/net/http/reqid"
)

// POST /create-control-program
func (a *API) createControlProgram(ctx context.Context, ins []struct {
	Type   string
	Params stdjson.RawMessage
}) interface{} {

	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			var (
				prog interface{}
				err  error
			)
			switch ins[i].Type {
			case "account":
				prog, err = a.createAccountControlProgram(subctx, ins[i].Params)
			default:
				err = errors.WithDetailf(httpjson.ErrBadRequest, "unknown control program type %q", ins[i].Type)
			}
			if err != nil {
				responses[i] = err
			} else {
				responses[i] = prog
			}
		}(i)
	}

	wg.Wait()
	return responses
}

func (a *API) createAccountControlProgram(ctx context.Context, input []byte) (interface{}, error) {
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
		acc, err := a.accounts.FindByAlias(ctx, parsed.AccountAlias)
		if err != nil {
			return nil, err
		}
		accountID = acc.ID
	}

	controlProgram, err := a.accounts.CreateControlProgram(ctx, accountID, false, time.Time{})
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"control_program": json.HexBytes(controlProgram),
	}
	return ret, nil
}
