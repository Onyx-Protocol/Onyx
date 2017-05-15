package core

import (
	"strings"

	chainjson "chain/encoding/json"
	"chain/protocol/ivy"
	"chain/protocol/vm"
)

type (
	compileReq struct {
		Contract string            `json:"contract"`
		Args     []ivy.ContractArg `json:"args"`
	}

	compileResp struct {
		Name    string              `json:"name"`
		Source  string              `json:"source"`
		Program chainjson.HexBytes  `json:"program"`
		Params  []ivy.ContractParam `json:"params"`
		Value   string              `json:"value"`
		Clauses []ivy.ClauseInfo    `json:"clause_info"`
		Opcodes string              `json:"opcodes"`
		Error   string              `json:"error"`
	}
)

func compileIvy(req compileReq) (compileResp, error) {
	var resp compileResp
	compiled, err := ivy.Compile(strings.NewReader(req.Contract), req.Args)
	if err == nil {
		resp.Name = compiled.Name
		resp.Source = req.Contract
		resp.Params = compiled.Params
		resp.Value = compiled.Value
		resp.Program = compiled.Program
		resp.Clauses = compiled.Clauses
		resp.Opcodes, err = vm.Disassemble(resp.Program, compiled.Labels)
		if err != nil {
			return resp, err
		}
	} else {
		resp.Error = err.Error()
	}
	return resp, nil
}
