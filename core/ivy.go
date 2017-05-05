package core

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"

	chainjson "chain/encoding/json"
	"chain/protocol/ivy"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

type (
	compileReq struct {
		Contract string        `json:"contract"`
		Args     []contractArg `json:"args"`
	}

	// exactly one of b, i, or s should be populated, the others should
	// be nil
	contractArg struct {
		b *bool               `json:"boolean,omitempty"`
		i *int64              `json:"integer,omitempty"`
		s *chainjson.HexBytes `json:"string,omitempty"`
	}

	compileResp struct {
		Source  string             `json:"source"`
		Program chainjson.HexBytes `json:"program"`
		Clauses []ivy.ClauseInfo   `json:"clause_info"`
		Opcodes string             `json:"opcodes"`
		Error   string             `json:"error"`
	}
)

func compileIvy(req compileReq) (compileResp, error) {
	fmt.Printf("* compileIvy:\n%s", spew.Sdump(req))
	var resp compileResp
	compiled, err := ivy.Compile(strings.NewReader(req.Contract))
	if err == nil {
		b := vmutil.NewBuilder()
		for _, a := range req.Args {
			switch {
			case a.b != nil:
				var n int64
				if *a.b {
					n = 1
				}
				b.AddInt64(n)
			case a.i != nil:
				b.AddInt64(*a.i)
			case a.s != nil:
				b.AddData(*a.s)
			}
		}
		resp.Source = req.Contract
		resp.Program, _ = b.Build() // error is impossible
		resp.Program = append(resp.Program, compiled.Program...)
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

func (a *contractArg) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	if r, ok := m["boolean"]; ok {
		var bval bool
		err = json.Unmarshal(r, &bval)
		if err != nil {
			return err
		}
		a.b = &bval
		return nil
	}
	if r, ok := m["integer"]; ok {
		var ival int64
		err = json.Unmarshal(r, &ival)
		if err != nil {
			return err
		}
		a.i = &ival
		return nil
	}
	r, ok := m["string"]
	if !ok {
		return fmt.Errorf("contract arg must define one of boolean, integer, string")
	}
	var sval chainjson.HexBytes
	err = json.Unmarshal(r, &sval)
	if err != nil {
		return err
	}
	a.s = &sval
	return nil
}
