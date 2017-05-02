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
		Bytes   chainjson.HexBytes `json:"bytes"`
		Opcodes string             `json:"opcodes"`
		Error   string             `json:"error"`
	}
)

func compileIvy(req compileReq) (compileResp, error) {
	var resp compileResp
	prog, err := ivy.Compile(strings.NewReader(req.Contract))
	if err == nil {
		resp.Bytes = prog
		resp.Opcodes, err = vm.Disassemble(prog)
		if err != nil {
			return resp, err
		}
	} else {
		resp.Error = err.Error()
	}
	return resp, nil
}
