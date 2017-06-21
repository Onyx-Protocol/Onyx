// +build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

func main() {
	fname := os.Getenv("CHAIN") + "/protocol/txvm/opcodes.go"
	fset := token.NewFileSet() // positions are relative to fset

	// Parse the file containing this very example
	// but stop after processing the imports.
	f, err := parser.ParseFile(fset, fname, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	fixed := map[string]bool{
		"BaseInt":  true,
		"NumOp":    true,
		"BaseData": true,
	}

	var constNum = 0

	for _, decl := range f.Decls {
		if d, ok := decl.(*ast.GenDecl); ok && d.Tok == token.CONST {
			for _, spec := range d.Specs {
				if s, ok := spec.(*ast.ValueSpec); ok {
					if val, ok := s.Values[0].(*ast.BasicLit); ok {
						if _, ok := fixed[s.Names[0].String()]; !ok {
							val.Value = fmt.Sprintf("%d", constNum)
							constNum++
						}
					}
				}
			}
		}
	}

	printer.Fprint(os.Stdout, fset, f)
}
