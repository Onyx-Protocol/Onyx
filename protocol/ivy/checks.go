package ivy

import "fmt"

func prohibitValueParams(contract *contract) error {
	for _, p := range contract.params {
		if p.typ == valueType {
			return fmt.Errorf("Value-typed contract parameter \"%s\" must appear in a \"locks\" clause", p.name)
		}
	}
	for _, c := range contract.clauses {
		for _, p := range c.params {
			if p.typ == valueType {
				return fmt.Errorf("Value-typed parameter \"%s\" of clause \"%s\" must appear in a \"requires\" clause", p.name, c.name)
			}
		}
	}
	return nil
}

func requireAllParamsUsedInClauses(params []*param, clauses []*clause) error {
	for _, p := range params {
		used := false
		for _, c := range clauses {
			err := requireAllParamsUsedInClause([]*param{p}, c)
			if err == nil {
				used = true
				break
			}
		}
		if !used {
			return fmt.Errorf("parameter \"%s\" is unused", p.name)
		}
	}
	return nil
}

func requireAllParamsUsedInClause(params []*param, clause *clause) error {
	for _, p := range params {
		used := false
		for _, stmt := range clause.statements {
			switch s := stmt.(type) {
			case *verifyStatement:
				used = references(s.expr, p.name)
			case *lockStatement:
				used = references(s.locked, p.name) || references(s.program, p.name)
			case *unlockStatement:
				used = references(s.expr, p.name)
			}
			if used {
				break
			}
		}
		if !used {
			for _, r := range clause.reqs {
				if references(r.amountExpr, p.name) || references(r.assetExpr, p.name) {
					used = true
					break
				}
			}
		}
		if !used {
			return fmt.Errorf("parameter \"%s\" is unused in clause \"%s\"", p.name, clause.name)
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
	case *call:
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

func requireAllValuesDisposedOnce(contract *contract, clause *clause) error {
	err := valueDisposedOnce(contract.value, clause)
	if err != nil {
		return err
	}
	for _, req := range clause.reqs {
		err = valueDisposedOnce(req.name, clause)
		if err != nil {
			return err
		}
	}
	return nil
}

func valueDisposedOnce(name string, clause *clause) error {
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
		return fmt.Errorf("value \"%s\" not disposed in clause \"%s\"", name, clause.name)
	case 1:
		return nil
	default:
		return fmt.Errorf("value \"%s\" disposed multiple times in clause \"%s\"", name, clause.name)
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

func assignIndexes(clause *clause) {
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

func typeCheckClause(contract *contract, clause *clause, env environ) error {
	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *verifyStatement:
			if t := stmt.expr.typ(env); t != boolType {
				return fmt.Errorf("expression in verify statement in clause \"%s\" has type \"%s\", must be Boolean", clause.name, t)
			}

		case *lockStatement:
			if t := stmt.locked.typ(env); t != valueType {
				return fmt.Errorf("expression in lock statement in clause \"%s\" has type \"%s\", must be Value", clause.name, t)
			}
			if t := stmt.program.typ(env); t != progType {
				return fmt.Errorf("program in lock statement in clause \"%s\" has type \"%s\", must be Program", clause.name, t)
			}

		case *unlockStatement:
			if t := stmt.expr.typ(env); t != valueType {
				return fmt.Errorf("expression \"%s\" in unlock statement of clause \"%s\" has type \"%s\", must be Value", stmt.expr, clause.name, t)
			}
			if stmt.expr.String() != contract.value {
				return fmt.Errorf("expression in unlock statement of clause \"%s\" must be the contract value", clause.name)
			}
		}
	}
	return nil
}

func typeCheckExpr(expr expression, env environ) error {
	switch e := expr.(type) {
	case *binaryExpr:
		lType := e.left.typ(env)
		rType := e.right.typ(env)

		if e.op.left != "" && lType != e.op.left {
			return fmt.Errorf("in \"%s\", left operand has type \"%s\", must be \"%s\"", e, lType, e.op.left)
		}
		if e.op.right != "" && rType != e.op.right {
			return fmt.Errorf("in \"%s\", right operand has type \"%s\", must be \"%s\"", e, rType, e.op.right)
		}

		switch e.op.op {
		case "==", "!=":
			if lType != rType {
				return fmt.Errorf("type mismatch in \"%s\": left operand has type \"%s\", right operand has type \"%s\"", e, lType, rType)
			}
			if lType == "Boolean" {
				return fmt.Errorf("in \"%s\": using \"%s\" on Boolean values not allowed", e, e.op.op)
			}
		}

	case *unaryExpr:
		if e.op.operand != "" && e.expr.typ(env) != e.op.operand {
			return fmt.Errorf("in \"%s\", operand has type \"%s\", must be \"%s\"", e, e.expr.typ(env), e.op.operand)
		}

	case *call:
		// handled in compileExpr
	}
	return nil
}
