package ivy

import (
	"fmt"
	"reflect"
)

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
		var e expression
		for _, stmt := range clause.statements {
			switch s := stmt.(type) {
			case *verifyStatement:
				e = s.expr
			case *outputStatement:
				e = s.call
			case *returnStatement:
				e = s.expr
			}
			if references(e, p.name) {
				used = true
				break
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
	case *propRef:
		return references(e.expr, name)
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

func requireValueParam(contract *contract) error {
	if len(contract.params) == 0 {
		return fmt.Errorf("must have at least one contract parameter")
	}
	if t := contract.params[len(contract.params)-1].typ; t != "Value" {
		return fmt.Errorf("final contract parameter has type \"%s\" but should be Value", t)
	}
	for i := 0; i < len(contract.params)-1; i++ {
		if contract.params[i].typ == "Value" {
			return fmt.Errorf("contract parameter %d has type Value, but only the final parameter may", i)
		}
	}
	return nil
}

func requireAllValuesDisposedOnce(contract *contract, clause *clause) error {
	err := paramDisposedOnce(contract.params[len(contract.params)-1], clause)
	if err != nil {
		return err
	}
	for _, p := range clause.params {
		if p.typ != "Value" {
			continue
		}
		err = paramDisposedOnce(p, clause)
		if err != nil {
			return err
		}
	}
	return nil
}

func paramDisposedOnce(p *param, clause *clause) error {
	var count int
	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *returnStatement:
			if references(stmt.expr, p.name) {
				count++
			}
		case *outputStatement:
			if len(stmt.call.args) == 1 && references(stmt.call.args[0], p.name) {
				count++
			}
		}
	}
	switch count {
	case 0:
		return fmt.Errorf("value parameter \"%s\" not disposed in clause \"%s\"", p.name, clause.name)
	case 1:
		return nil
	default:
		return fmt.Errorf("value parameter \"%s\" disposed multiple times in clause \"%s\"", p.name, clause.name)
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

func decorateOutputs(contract *contract, clause *clause, env environ) error {
	for _, s := range clause.statements {
		stmt, ok := s.(*outputStatement)
		if !ok {
			continue
		}
		if t := stmt.call.fn.typ(env); t != progType {
			return fmt.Errorf("type of function (%s) in output statement of clause \"%s\" is \"%s\", must be %s", stmt.call.fn, clause.name, t, progType)
		}
		if len(stmt.call.args) != 1 {
			return fmt.Errorf("not yet supported: zero or multiple arguments in call to \"%s\" in output statement of clause \"%s\"", stmt.call.fn, clause.name)
		}
		valueExpr := stmt.call.args[0]
		if t := valueExpr.typ(env); t != "Value" {
			return fmt.Errorf("not yet supported: argument of non-Value type \"%s\" passed to \"%s\" in output statement of clause \"%s\"", t, stmt.call.fn, clause.name)
		}
		if valueVar, ok := valueExpr.(varRef); ok {
			if entry, ok := env[string(valueVar)]; ok {
				if entry.r == roleContractParam {
					// Contract value doesn't have to be matched against an
					// AssetAmount
					continue
				}
			}
		}
		// Look for a verify statement matching valueExpr to an
		// assetamount.
		found := false
		for _, s2 := range clause.statements {
			v, ok := s2.(*verifyStatement)
			if !ok {
				continue
			}
			if v.associatedOutput != nil {
				// This verify is already associated with a different output
				// statement.
				continue
			}
			e, ok := v.expr.(*binaryExpr)
			if !ok {
				continue
			}
			if e.op.op != "==" {
				continue
			}

			// Check that e.left is the value param + ".assetAmount" and e.right is an
			// assetamount param, or vice versa.
			var other expression
			check := func(e expression) bool {
				if prop, ok := e.(*propRef); ok {
					return reflect.DeepEqual(prop.expr, valueExpr) && prop.property == "assetAmount"
				}
				return false
			}

			if check(e.left) {
				other = e.right
			} else if check(e.right) {
				other = e.left
			} else {
				continue
			}
			if other.typ(env) != "AssetAmount" {
				continue
			}
			v.associatedOutput = stmt
			stmt.assetAmount = other
			found = true
			break
		}
		if !found {
			return fmt.Errorf("Value expression \"%s\" is in an output statement in clause \"%s\" but not checked in a verify statement", valueExpr, clause.name)
		}
	}
	return nil
}

func assignIndexes(clause *clause) {
	var nextIndex int64
	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *outputStatement:
			stmt.index = nextIndex
			nextIndex++

		case *returnStatement:
			nextIndex++
		}
	}
}

func typeCheckClause(contract *contract, clause *clause, env environ) error {
	for i, s := range clause.statements {
		switch stmt := s.(type) {
		case *verifyStatement:
			if stmt.associatedOutput != nil {
				// This verify is associated with an output. It doesn't get
				// compiled; instead it contributes its terms to the output
				// statement's CHECKOUTPUT.
				continue
			}
			if t := stmt.expr.typ(env); t != "Boolean" {
				return fmt.Errorf("expression in verify statement in clause \"%s\" has type \"%s\", must be Boolean", clause.name, t)
			}

		case *returnStatement:
			if i != len(clause.statements)-1 {
				return fmt.Errorf("return must be the final statement of clause \"%s\"", clause.name)
			}
			if t := stmt.expr.typ(env); t != "Value" {
				return fmt.Errorf("expression \"%s\" in return statement of clause \"%s\" has type \"%s\", must be Value", stmt.expr, clause.name, t)
			}
			if !references(stmt.expr, contract.params[len(contract.params)-1].name) {
				return fmt.Errorf("expression in return statement of clause \"%s\" must be the contract Value parameter", clause.name)
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
		b := referencedBuiltin(e.fn)
		if b == nil {
			return fmt.Errorf("unknown function \"%s\"", e.fn)
		}
		if len(e.args) != len(b.args) {
			return fmt.Errorf("wrong number of args for \"%s\": have %d, want %d", b.name, len(e.args), len(b.args))
		}
		for i, actual := range e.args {
			if b.args[i] != "" && actual.typ(env) != b.args[i] {
				return fmt.Errorf("argument %d to \"%s\" has type \"%s\", must be \"%s\"", i, b.name, actual.typ(env), b.args[i])
			}
		}
	}
	return nil
}
