package core

import (
	"strings"

	chainjson "chain/encoding/json"
	"chain/protocol/ivy"
)

type (
	compileReq struct {
		Source string                       `json:"source"`
		ArgMap map[string][]ivy.ContractArg `json:"arg_map"`
	}

	compileResp struct {
		Contracts []*ivy.Contract               `json:"contracts,omitempty"`
		Programs  map[string]chainjson.HexBytes `json:"program_map,omitempty"`
		Error     string                        `json:"error,omitempty"`
	}
)

func compileIvy(req compileReq) compileResp {
	contracts, err := ivy.Compile(strings.NewReader(req.Source))
	if err != nil {
		return compileResp{Error: err.Error()}
	}
	var m map[string]chainjson.HexBytes
	for _, contract := range contracts {
		if args, ok := req.ArgMap[contract.Name]; ok {
			prog, err := ivy.Instantiate(contract.Body, contract.Params, contract.Recursive, args)
			if err != nil {
				return compileResp{Error: err.Error()}
			}
			if m == nil {
				m = make(map[string]chainjson.HexBytes)
			}
			m[contract.Name] = prog
		}
	}

	return compileResp{Contracts: compiled, Programs: m}
}
