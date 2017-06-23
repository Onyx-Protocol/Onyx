package compiler

import "fmt"

func checkRecursive(contract *Contract) bool {
	for _, clause := range contract.Clauses {
		for _, stmt := range clause.Statements {
			if l, ok := stmt.(*LockStatement); ok {
				if c, ok := l.Program.(*CallExpr); ok {
					if references(c.fn, contract.Name) {
						return true
					}
				}
			}
		}
	}
	return false
}

func prohibitSigParams(contract *Contract) error {
	for _, p := range contract.Params {
		if p.Type == sigType {
			return fmt.Errorf("contract parameter \"%s\" has type Signature, but contract parameters cannot have type Signature", p.Name)
		}
	}
	return nil
}

func prohibitValueParams(contract *Contract) error {
	for _, p := range contract.Params {
		if p.Type == valueType {
			return fmt.Errorf("Value-typed contract parameter \"%s\" must appear in a \"locks\" clause", p.Name)
		}
	}
	for _, c := range contract.Clauses {
		for _, p := range c.Params {
			if p.Type == valueType {
				return fmt.Errorf("Value-typed parameter \"%s\" of clause \"%s\" must appear in a \"requires\" clause", p.Name, c.Name)
			}
		}
	}
	return nil
}

func requireAllParamsUsedInClauses(params []*Param, clauses []*Clause) error {
	for _, p := range params {
		used := false
		for _, c := range clauses {
			err := requireAllParamsUsedInClause([]*Param{p}, c)
			if err == nil {
				used = true
				break
			}
		}
		if !used {
			return fmt.Errorf("parameter \"%s\" is unused", p.Name)
		}
	}
	return nil
}

func requireAllParamsUsedInClause(params []*Param, clause *Clause) error {
	for _, p := range params {
		used := false
		for _, stmt := range clause.Statements {
			switch s := stmt.(type) {
			case *VerifyStatement:
				used = references(s.Expr, p.Name)
			case *LockStatement:
				used = references(s.Locked, p.Name) || references(s.Program, p.Name)
			case *UnlockStatement:
				used = references(s.Expr, p.Name)
			}
			if used {
				break
			}
		}
		if !used {
			for _, r := range clause.Reqs {
				if references(r.amountExpr, p.Name) || references(r.assetExpr, p.Name) {
					used = true
					break
				}
			}
		}
		if !used {
			return fmt.Errorf("parameter \"%s\" is unused in clause \"%s\"", p.Name, clause.Name)
		}
	}
	return nil
}

func references(expr Expression, name string) bool {
	switch e := expr.(type) {
	case *BinaryExpr:
		return references(e.left, name) || references(e.right, name)
	case *UnaryExpr:
		return references(e.expr, name)
	case *CallExpr:
		if references(e.fn, name) {
			return true
		}
		for _, a := range e.args {
			if references(a, name) {
				return true
			}
		}
		return false
	case VarRef:
		return string(e) == name
	case ListExpr:
		for _, elt := range []Expression(e) {
			if references(elt, name) {
				return true
			}
		}
		return false
	}
	return false
}

func requireAllValuesDisposedOnce(contract *Contract, clause *Clause) error {
	err := valueDisposedOnce(contract.Value, clause)
	if err != nil {
		return err
	}
	for _, req := range clause.Reqs {
		err = valueDisposedOnce(req.Name, clause)
		if err != nil {
			return err
		}
	}
	return nil
}

func valueDisposedOnce(name string, clause *Clause) error {
	var count int
	for _, s := range clause.Statements {
		switch stmt := s.(type) {
		case *UnlockStatement:
			if references(stmt.Expr, name) {
				count++
			}
		case *LockStatement:
			if references(stmt.Locked, name) {
				count++
			}
		}
	}
	switch count {
	case 0:
		return fmt.Errorf("value \"%s\" not disposed in clause \"%s\"", name, clause.Name)
	case 1:
		return nil
	default:
		return fmt.Errorf("value \"%s\" disposed multiple times in clause \"%s\"", name, clause.Name)
	}
}

func referencedBuiltin(expr Expression) *builtin {
	if v, ok := expr.(VarRef); ok {
		for _, b := range builtins {
			if string(v) == b.name {
				return &b
			}
		}
	}
	return nil
}

func assignIndexes(clause *Clause) {
	var nextIndex int64
	for _, s := range clause.Statements {
		switch stmt := s.(type) {
		case *LockStatement:
			stmt.index = nextIndex
			nextIndex++

		case *UnlockStatement:
			nextIndex++
		}
	}
}

func typeCheckClause(contract *Contract, clause *Clause, env *environ) error {
	for _, s := range clause.Statements {
		switch stmt := s.(type) {
		case *VerifyStatement:
			if t := stmt.Expr.typ(env); t != boolType {
				return fmt.Errorf("expression in verify statement in clause \"%s\" has type \"%s\", must be Boolean", clause.Name, t)
			}

		case *LockStatement:
			if t := stmt.Locked.typ(env); t != valueType {
				return fmt.Errorf("expression in lock statement in clause \"%s\" has type \"%s\", must be Value", clause.Name, t)
			}
			if t := stmt.Program.typ(env); t != progType {
				return fmt.Errorf("program in lock statement in clause \"%s\" has type \"%s\", must be Program", clause.Name, t)
			}

		case *UnlockStatement:
			if t := stmt.Expr.typ(env); t != valueType {
				return fmt.Errorf("expression \"%s\" in unlock statement of clause \"%s\" has type \"%s\", must be Value", stmt.Expr, clause.Name, t)
			}
			if stmt.Expr.String() != contract.Value {
				return fmt.Errorf("expression in unlock statement of clause \"%s\" must be the contract value", clause.Name)
			}
		}
	}
	return nil
}
