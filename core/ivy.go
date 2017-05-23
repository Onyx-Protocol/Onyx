package core

import (
	"strings"

	"chain/protocol/ivy"
)

type (
	compileReq struct {
		Contract string            `json:"contract"`
		Args     []ivy.ContractArg `json:"args"`
	}

	compileResp struct {
		Contracts []*ivy.Contract `json:"contracts,omitempty"`
		Error     string          `json:"error,omitempty"`
	}
)

func compileIvy(req compileReq) compileResp {
	compiled, err := ivy.Compile(strings.NewReader(req.Contract), req.Args)
	if err != nil {
		return compileResp{Error: err.Error()}
	}
	return compileResp{Contracts: compiled}
}
