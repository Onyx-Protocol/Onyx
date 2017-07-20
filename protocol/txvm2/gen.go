// +build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

func main() {
	ops()
	types()
}

func ops() {
	ops := getOps()
	opinfoName := txvmFile("opinfo.go")
	out, err := os.Create(opinfoName)
	must(err)
	fmt.Fprint(out, "// Auto-generated from ops.go by gen.go\n\npackage txvm2\n\n")

	fmt.Fprint(out, "var opNames = [...]string{\n")
	for _, op := range ops {
		fmt.Fprintf(out, "\t%s: \"%s\",\n", op, strings.ToLower(op[2:]))
	}
	fmt.Fprint(out, "}\n\n")

	fmt.Fprint(out, "var opCodes = map[string]byte{\n")
	for _, op := range ops {
		fmt.Fprintf(out, "\t\"%s\": %s,\n", strings.ToLower(op[2:]), op)
	}
	fmt.Fprint(out, "}\n\n")

	fmt.Fprint(out, "var opFuncs = [...]func(*vm){\n")
	for _, op := range ops {
		switch op {
		case "OpCommand", "OpSatisfy", "OpProveAssetRange":
			// do nothing - avoid initialization loop
		default:
			fmt.Fprintf(out, "\t%s: %c%s,\n", op, unicode.ToLower(rune(op[0])), op[1:])
		}
	}
	fmt.Fprint(out, "}\n\n")

	out.Close()

	cmd := exec.Command("gofmt", "-w", opinfoName)
	must(cmd.Run())
}

func getOps() []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, txvmFile("ops.go"), nil, 0)
	must(err)
	var constDecl *ast.GenDecl
	for _, d := range f.Decls {
		if gendecl, ok := d.(*ast.GenDecl); ok && gendecl.Tok == token.CONST {
			constDecl = gendecl
			break
		}
	}
	if constDecl == nil {
		panic("ops.go has no top-level const declaration")
	}
	var ops []string
	for _, spec := range constDecl.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			panic("const decl contains non-const values?!")
		}
		if len(vspec.Names) != 1 {
			panic(fmt.Errorf("const spec contains %d names, want 1", len(vspec.Names)))
		}
		name := vspec.Names[0].Name
		if name == "Op0" {
			continue
		}
		if !strings.HasPrefix(name, "Op") {
			continue
		}
		ops = append(ops, name)
	}
	return ops
}

func types() {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, txvmFile("types.go"), nil, 0)
	must(err)
	typeinfoName := txvmFile("typeinfo.go")
	out, err := os.Create(typeinfoName)
	must(err)

	predefined := make(map[[2]string]bool) // set of [type,method] pairs

	for _, d := range f.Decls {
		if f, ok := d.(*ast.FuncDecl); ok {
			if f.Recv == nil {
				continue
			}
			if len(f.Recv.List) != 1 {
				panic(fmt.Errorf("a method receiver with %d receivers??", len(f.Recv.List)))
			}
			recvType := f.Recv.List[0].Type
			var recvTypeName string
			switch rt := recvType.(type) {
			case *ast.Ident:
				recvTypeName = rt.Name
			case *ast.StarExpr:
				if n, ok := rt.X.(*ast.Ident); ok {
					recvTypeName = n.Name
				}
			}
			if recvTypeName == "" {
				continue
			}
			predefined[[2]string{recvTypeName, f.Name.Name}] = true
		}
	}

	fmt.Fprint(out, "// Auto-generated from types.go by gen.go\n\npackage txvm2\n\n")

	for _, d := range f.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.TYPE {
			for _, s := range g.Specs {
				ts := s.(*ast.TypeSpec)
				if st, ok := ts.Type.(*ast.StructType); ok {
					nFields := 0
					for _, f := range st.Fields.List {
						nFields += len(f.Names)
					}

					if !predefined[[2]string{ts.Name.Name, "entuple"}] {
						fmt.Fprintf(out, "func (x %s) entuple() tuple {\n", ts.Name.Name)
						fmt.Fprintf(out, "\treturn tuple{\n")
						fmt.Fprintf(out, "\t\tvbytes(\"%s\"),\n", ts.Name.Name)
						for _, f := range st.Fields.List {
							for _, name := range f.Names {
								switch {
								case isItem(f.Type):
									fmt.Fprintf(out, "\t\tx.%s,\n", name.Name)
								case isInt64(f.Type):
									fmt.Fprintf(out, "\t\tvint64(x.%s),\n", name.Name)
								case isBytes(f.Type):
									fmt.Fprintf(out, "\t\tvbytes(x.%s),\n", name.Name)
								default:
									fmt.Fprintf(out, "\t\tx.%s.entuple(),\n", name.Name)
								}
							}
						}
						fmt.Fprintf(out, "\t}\n")
						fmt.Fprintf(out, "}\n\n")
					}

					if !predefined[[2]string{ts.Name.Name, "detuple"}] {
						fmt.Fprintf(out, "func (x *%s) detuple(t tuple) bool {\n", ts.Name.Name)
						fmt.Fprintf(out, "\tif len(t) != %d { return false }\n", nFields+1)
						fmt.Fprintf(out, "\tif n, ok := t[0].(vbytes); !ok || string(n) != \"%s\" { return false }\n", ts.Name.Name)
						i := 1
						for _, f := range st.Fields.List {
							for _, name := range f.Names {
								switch {
								case isItem(f.Type):
									fmt.Fprintf(out, "\tx.%s = t[%d]\n", name.Name, i)
								case isInt64(f.Type):
									fmt.Fprintf(out, "\tx.%s = int64(t[%d].(vint64))\n", name.Name, i)
								case isBytes(f.Type):
									fmt.Fprintf(out, "\tx.%s = []byte(t[%d].(vbytes))\n", name.Name, i)
								default:
									fmt.Fprintf(out, "\tif !x.%s.detuple(t[%d].(tuple)) { return false }\n", name.Name, i)
								}
								i++
							}
						}
						fmt.Fprintf(out, "\treturn true\n")
						fmt.Fprintf(out, "}\n\n")
					}

					popName := fmt.Sprintf("pop%s", strings.Title(ts.Name.Name))
					if !predefined[[2]string{"vm", popName}] {
						fmt.Fprintf(out, "func (vm *vm) %s(stacknum int) %s {\n", popName, ts.Name.Name)
						fmt.Fprintf(out, "\tv := vm.pop(stacknum)\n")
						fmt.Fprintf(out, "\tt := v.(tuple)\n")
						fmt.Fprintf(out, "\tvar x %s\n", ts.Name.Name)
						fmt.Fprintf(out, "\tif !x.detuple(t) { panic(\"tuple is not a valid %s\") }\n", ts.Name.Name)
						fmt.Fprintf(out, "\treturn x\n")
						fmt.Fprintf(out, "}\n\n")
					}

					pushName := fmt.Sprintf("push%s", strings.Title(ts.Name.Name))
					if !predefined[[2]string{"vm", pushName}] {
						fmt.Fprintf(out, "func (vm *vm) push%s(stacknum int, x %s) {\n", strings.Title(ts.Name.Name), ts.Name.Name)
						fmt.Fprintf(out, "\tvm.push(stacknum, x.entuple())\n")
						fmt.Fprintf(out, "}\n\n")
					}
				}
			}
		}
	}

	out.Close()

	cmd := exec.Command("gofmt", "-w", typeinfoName)
	must(cmd.Run())
}

func isItem(t ast.Expr) bool {
	if id, ok := t.(*ast.Ident); ok {
		return id.Name == "item"
	}
	return false
}

func isInt64(t ast.Expr) bool {
	if id, ok := t.(*ast.Ident); ok {
		return id.Name == "int64"
	}
	return false
}

func isBytes(t ast.Expr) bool {
	if a, ok := t.(*ast.ArrayType); ok {
		if a.Len != nil {
			return false
		}
		if id, ok := a.Elt.(*ast.Ident); ok {
			return id.Name == "byte"
		}
	}
	return false
}

func txvmFile(name string) string {
	return os.Getenv("CHAIN") + "/protocol/txvm2/" + name
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
