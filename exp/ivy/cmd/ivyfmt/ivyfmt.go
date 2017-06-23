package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"chain/exp/ivy/compiler"
)

func main() {
	b, err := ioutil.ReadAll(os.Stdin)
	must(err)
	contracts, err := compiler.Parse(b)
	must(err)
	firstContract := true
	for _, contract := range contracts {
		if firstContract {
			firstContract = false
		} else {
			fmt.Print("\n")
		}
		fmt.Printf("contract %s(", contract.Name)
		printParams(contract.Params)
		fmt.Printf(") locks %s {\n", contract.Value)
		for _, clause := range contract.Clauses {
			fmt.Printf("\tclause %s(", clause.Name)
			printParams(clause.Params)
			fmt.Print(")")
			if len(clause.Reqs) > 0 {
				fmt.Print(" requires ")
				firstClause := true
				for _, req := range clause.Reqs {
					if firstClause {
						firstClause = false
					} else {
						fmt.Print(", ")
					}
					fmt.Printf("%s: %s of %s", req.Name, req.Amount, req.Asset)
				}
			}
			fmt.Print("{\n")
			for _, s := range clause.Statements {
				switch stmt := s.(type) {
				case *compiler.VerifyStatement:
					fmt.Print("\t\tverify ")
					printExpr(stmt.Expr)
				case *compiler.LockStatement:
					fmt.Print("\t\tlock ")
					printExpr(stmt.Locked)
					fmt.Print(" with ")
					printExpr(stmt.Program)
				case *compiler.UnlockStatement:
					fmt.Print("\t\tunlock ")
					printExpr(stmt.Expr)
				}
				fmt.Print("\n")
			}
			fmt.Print("\t}\n")
		}
		fmt.Print("}\n")
	}
}

func printParams(params []*compiler.Param) {
	first := true
	for i := 0; i < len(params); i++ {
		if first {
			first = false
		} else {
			fmt.Print(", ")
		}
		fmt.Print(params[i].Name)
		for i < len(params)-1 {
			if params[i+1].Type == params[i].Type {
				i++
				fmt.Printf(", %s", params[i].Name)
			}
		}
		fmt.Printf(" %s", params[i].Type)
	}
}

func printExpr(e compiler.Expression) {
	switch expr := e.(type) {
	case compiler.BinaryExpr:

	case compiler.UnaryExpr:
	case compiler.CallExpr:
	case compiler.VarRef:
	case compiler.BytesLiteral:
	case compiler.IntegerLiteral:
	case compiler.BooleanLiteral:
	case compiler.ListExpr:
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
