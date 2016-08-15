package core

import (
	"encoding/json"
	"sync"

	"golang.org/x/net/context"

	"chain/core/account"
	"chain/errors"
	"chain/net/http/httpjson"
)

// POST /create-control-program
func createControlProgram(ctx context.Context, ins []struct {
	Type       string
	Parameters json.RawMessage
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
				prog, err = createAccountControlProgram(ctx, ins[i].Parameters)
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

func createAccountControlProgram(ctx context.Context, input []byte) (interface{}, error) {
	var parsed struct {
		AccountID string `json:"account_id"`
	}
	err := json.Unmarshal(input, &parsed)
	if err != nil {
		return nil, errors.WithDetailf(httpjson.ErrBadRequest, "no 'account_id' parameter sent")
	}

	controlProgram, err := account.CreateControlProgram(ctx, parsed.AccountID)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"control_program": controlProgram,
	}
	return ret, nil
}
