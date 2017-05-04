package core

import (
	"strings"

	chainjson "chain/encoding/json"
	"chain/protocol/ivy"
	"chain/protocol/vm"
)

type (
	compileReq struct {
		Contract string `json:"contract"`
	}

	compileResp struct {
		Program chainjson.HexBytes `json:"program"`
		Clauses []ivy.ClauseInfo   `json:"clause_info"`
		Opcodes string             `json:"opcodes"`
		Error   string             `json:"error"`
	}
)

func compileIvy(req compileReq) (compileResp, error) {
	var resp compileResp
	compiled, err := ivy.Compile(strings.NewReader(req.Contract))
	if err == nil {
		resp.Program = compiled.Program
		resp.Clauses = compiled.Clauses
		resp.Opcodes, err = vm.Disassemble(resp.Program)
		if err != nil {
			return resp, err
		}
	} else {
		resp.Error = err.Error()
	}
	return resp, nil
}
