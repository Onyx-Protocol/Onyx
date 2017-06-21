// +build ignore

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
)

func main() {
	opcodes := getOpcodes()
	var lower []string
	for _, op := range opcodes {
		lower = append(lower, strings.ToLower(op))
	}

	tpl.Execute(os.Stdout, struct {
		OpCodes []string
		Lower   []string
	}{opcodes, lower})
}

func getOpcodes() []string {
	fset := token.NewFileSet() // positions are relative to fset

	// Parse the file containing this very example
	// but stop after processing the imports.
	f, err := parser.ParseFile(fset, os.Getenv("CHAIN")+"/protocol/txvm/opcodes.go", nil, 0)
	if err != nil {
		panic(err)
	}

	var opcodes []string
	exclude := map[string]bool{
		"NumOp":    true,
		"BaseInt":  true,
		"BaseData": true,
	}

	for _, decl := range f.Decls {
		if d, ok := decl.(*ast.GenDecl); ok && d.Tok == token.CONST {
			for _, spec := range d.Specs {
				if s, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range s.Names {
						if exclude[name.String()] {
							continue
						}
						opcodes = append(opcodes, name.String())
					}
				}
			}
		}
	}

	return opcodes
}

var tpl = template.Must(template.New("").Parse(`package txvm

//go:generate sh gen.sh

var OpNames = [...]string{
{{- with $top := .}}
{{- range $index, $name := .OpCodes }}
	{{$name}}: "{{index $top.Lower $index}}",
{{- end }}
{{- end}}
}

var OpCodes = map[string]byte{
	{{- with $top := .}}
	{{- range $index, $name := .OpCodes }}
		"{{index $top.Lower $index}}": {{$name}},
	{{- end }}
	{{- end}}
}
`))
