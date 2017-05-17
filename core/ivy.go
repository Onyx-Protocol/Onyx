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
		Name           string              `json:"name"`
		Source         string              `json:"source"`
		Body           chainjson.HexBytes  `json:"body"`
		Program        chainjson.HexBytes  `json:"program"`
		Params         []ivy.ContractParam `json:"params"`
		Value          string              `json:"value"`
		Clauses        []ivy.ClauseInfo    `json:"clause_info"`
		BodyOpcodes    string              `json:"body_opcodes"`
		ProgramOpcodes string              `json:"program_opcodes"`
		Error          string              `json:"error"`
	}
)

func compileIvy(req compileReq) (compileResp, error) {
	var resp compileResp
	compiled, err := ivy.Compile(strings.NewReader(req.Contract), &req.Args)
	if err == nil {
		resp.Name = compiled.Name
		resp.Source = req.Contract
		resp.Params = compiled.Params
		resp.Value = compiled.Value
		resp.Body = compiled.Body
		resp.Program = compiled.Program
		resp.Clauses = compiled.Clauses
		resp.BodyOpcodes, err = vm.Disassemble(resp.Body, compiled.Labels)
		if err != nil {
			return resp, err
		}
		resp.ProgramOpcodes, err = vm.Disassemble(resp.Program, compiled.Labels)
		if err != nil {
			return resp, err
		}
	} else {
		resp.Error = err.Error()
	}
	return resp, nil
}
