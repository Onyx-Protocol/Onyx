package core

import (
	"strings"

	"chain/protocol/ivy"
)

type (
	compileReq struct {
		Source string                       `json:"source"`
		ArgMap map[string][]ivy.ContractArg `json:"arg_map"`
	}

	compileResp struct {
		Contracts []*ivy.Contract `json:"contracts,omitempty"`
		Error     string          `json:"error,omitempty"`
	}
)

func compileIvy(req compileReq) compileResp {
	compiled, err := ivy.Compile(strings.NewReader(req.Source), req.ArgMap)
	if err != nil {
		return compileResp{Error: err.Error()}
	}
	return compileResp{Contracts: compiled}
}
