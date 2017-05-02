package ivy

import "fmt"

func prohibitDuplicateClauseNames(contract *contract) error {
	// Prohibit duplicate clause names
	for i, c := range contract.clauses {
		for j := i + 1; j < len(contract.clauses); j++ {
			if c.name == contract.clauses[j].name {
				return fmt.Errorf("clause name %s is duplicated", c.name)
			}
		}
	}
	return nil
}

func prohibitDuplicateVars(contract *contract) error {
	for i, p := range contract.params {
		for j := i + 1; j < len(contract.params); j++ {
			if p.name == contract.params[j].name {
				return fmt.Errorf("contract parameter %s is duplicated", p.name)
			}
		}
	}
	for _, clause := range contract.clauses {
		for _, clauseParam := range clause.params {
			for _, contractParam := range contract.params {
				if clauseParam.name == contractParam.name {
					return fmt.Errorf("parameter %s in clause %s shadows contract parameter", clauseParam.name, clause.name)
				}
			}
		}
	}
	return nil
}

func requireValueParam(contract *contract) error {
	if len(contract.params) == 0 {
		return fmt.Errorf("must have at least one contract parameter")
	}
	if contract.params[len(contract.params)-1].typ != "Value" {
		return fmt.Errorf("final contract parameter has type %s but should be Value", contract.params[len(contract.params)-1].typ)
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
			if exprReferencesParam(stmt.expr, p) {
				count++
			}
		case *outputStatement:
			if len(stmt.call.args) == 1 && exprReferencesParam(stmt.call.args[0], p) {
				count++
			}
		}
	}
	switch count {
	case 0:
		return fmt.Errorf("value parameter %s not disposed in clause %s", p.name, clause.name)
	case 1:
		return nil
	default:
		return fmt.Errorf("value parameter %s disposed multiple times in clause %s", p.name, clause.name)
	}
}

func exprReferencesParam(e expression, p *param) bool {
	if r, ok := e.(*ref); ok {
		return len(r.names) == 1 && r.names[0] == p.name
	}
	return false
}

func decorateRefs(contract *contract, clause *clause) error {
	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *verifyStatement:
			err := decorateRefsInExpr(contract, clause, stmt.expr)
			if err != nil {
				return err
			}

		case *outputStatement:
			err := decorateRefsInExpr(contract, clause, stmt.call)
			if err != nil {
				return err
			}

		case *returnStatement:
			err := decorateRefsInExpr(contract, clause, stmt.expr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func decorateRefsInExpr(contract *contract, clause *clause, expr expression) error {
	switch e := expr.(type) {
	case *binaryExpr:
		err := decorateRefsInExpr(contract, clause, e.left)
		if err != nil {
			return err
		}
		err = decorateRefsInExpr(contract, clause, e.right)
		return err

	case *unaryExpr:
		return decorateRefsInExpr(contract, clause, e.expr)

	case *call:
		err := decorateRefsInExpr(contract, clause, e.fn)
		if err != nil {
			return err
		}
		for _, a := range e.args {
			err = decorateRefsInExpr(contract, clause, a)
			if err != nil {
				return err
			}
		}

	case *ref:
		refStr := e.String()
		for _, b := range builtins {
			if refStr == b.name {
				e.builtin = b
				return nil
			}
		}
		for _, p := range contract.params {
			if e.names[0] == p.name {
				e.param = p
				return nil
			}
		}
		for _, p := range clause.params {
			if e.names[0] == p.name {
				e.param = p
				return nil
			}
		}
		return fmt.Errorf("undefined variable %s", e.names[0])
	}
	return nil
}

func decorateOutputs(contract *contract, clause *clause) error {
	for _, s := range clause.statements {
		stmt, ok := s.(*outputStatement)
		if !ok {
			continue
		}
		if len(stmt.call.args) != 1 {
			return fmt.Errorf("multiple arguments in output function calls not yet supported")
		}
		r, ok := stmt.call.args[0].(*ref)
		if !ok {
			return fmt.Errorf("passing anything other than a value parameter to an output function call not yet supported")
		}
		if r.param == nil || r.param.typ != "Value" {
			return fmt.Errorf("%s is not a value parameter", r)
		}

		if r.param.name == contract.params[len(contract.params)-1].name {
			// It's the contract value param and doesn't have to be matched
			// against an AssetAmount parameter.
			continue
		}

		// Look for a verify statement matching this ref to an assetamount
		// parameter.
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
			if e.op != "==" {
				continue
			}

			// Check that e.left is the value param + ".assetAmount" and e.right is an
			// assetamount param, or vice versa.
			var other expression
			check := func(e expression) bool {
				if r2, ok := e.(*ref); ok {
					return len(r2.names) == 2 && r2.names[0] == r.param.name && r2.names[1] == "assetAmount"
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
			otherRef, ok := other.(*ref)
			if !ok {
				continue
			}
			if otherRef.param == nil || otherRef.param.typ != "AssetAmount" {
				continue
			}
			v.associatedOutput = stmt
			stmt.param = otherRef.param
			found = true
			break
		}
		if !found {
			return fmt.Errorf("value param %s is in an output statement but not checked in a verify statement", r)
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
