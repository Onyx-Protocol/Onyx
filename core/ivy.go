package core

import (
	"strings"

	chainjson "chain/encoding/json"
	"chain/protocol/ivy"
)

type (
	compileReq struct {
		Contract string `json:"contract"`
	}

	compileResp struct {
		Program chainjson.HexBytes `json:"program"`
		OK      bool
		Error   string
	}
)

func compileIvy(req compileReq) compileResp {
	prog, err := ivy.Compile(strings.NewReader(req.Contract))
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	return compileResp{
		Program: prog,
		OK:      err == nil,
		Error:   errStr,
	}
}
