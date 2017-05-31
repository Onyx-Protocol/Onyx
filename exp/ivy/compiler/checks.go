package compiler

import "fmt"

func checkRecursive(contract *Contract) bool {
	for _, clause := range contract.Clauses {
		for _, stmt := range clause.statements {
			if l, ok := stmt.(*lockStatement); ok {
				if c, ok := l.program.(*callExpr); ok {
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
		for _, stmt := range clause.statements {
			switch s := stmt.(type) {
			case *verifyStatement:
				used = references(s.expr, p.Name)
			case *lockStatement:
				used = references(s.locked, p.Name) || references(s.program, p.Name)
			case *unlockStatement:
				used = references(s.expr, p.Name)
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

func references(expr expression, name string) bool {
	switch e := expr.(type) {
	case *binaryExpr:
		return references(e.left, name) || references(e.right, name)
	case *unaryExpr:
		return references(e.expr, name)
	case *callExpr:
		if references(e.fn, name) {
			return true
		}
		for _, a := range e.args {
			if references(a, name) {
				return true
			}
		}
		return false
	case varRef:
		return string(e) == name
	case listExpr:
		for _, elt := range []expression(e) {
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
	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *unlockStatement:
			if references(stmt.expr, name) {
				count++
			}
		case *lockStatement:
			if references(stmt.locked, name) {
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

func referencedBuiltin(expr expression) *builtin {
	if v, ok := expr.(varRef); ok {
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
	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *lockStatement:
			stmt.index = nextIndex
			nextIndex++

		case *unlockStatement:
			nextIndex++
		}
	}
}

func typeCheckClause(contract *Contract, clause *Clause, env *environ) error {
	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *verifyStatement:
			if t := stmt.expr.typ(env); t != boolType {
				return fmt.Errorf("expression in verify statement in clause \"%s\" has type \"%s\", must be Boolean", clause.Name, t)
			}

		case *lockStatement:
			if t := stmt.locked.typ(env); t != valueType {
				return fmt.Errorf("expression in lock statement in clause \"%s\" has type \"%s\", must be Value", clause.Name, t)
			}
			if t := stmt.program.typ(env); t != progType {
				return fmt.Errorf("program in lock statement in clause \"%s\" has type \"%s\", must be Program", clause.Name, t)
			}

		case *unlockStatement:
			if t := stmt.expr.typ(env); t != valueType {
				return fmt.Errorf("expression \"%s\" in unlock statement of clause \"%s\" has type \"%s\", must be Value", stmt.expr, clause.Name, t)
			}
			if stmt.expr.String() != contract.Value {
				return fmt.Errorf("expression in unlock statement of clause \"%s\" must be the contract value", clause.Name)
			}
		}
	}
	return nil
}
