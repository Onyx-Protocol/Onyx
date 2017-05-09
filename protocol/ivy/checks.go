package ivy

import "fmt"

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
			if exprReferencesParam(e, p) {
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

func exprReferencesParam(expr expression, p *param) bool {
	switch e := expr.(type) {
	case *binaryExpr:
		return exprReferencesParam(e.left, p) || exprReferencesParam(e.right, p)
	case *unaryExpr:
		return exprReferencesParam(e.expr, p)
	case *call:
		if exprReferencesParam(e.fn, p) {
			return true
		}
		for _, a := range e.args {
			if exprReferencesParam(a, p) {
				return true
			}
		}
		return false
	case *propRef:
		return exprReferencesParam(e.expr, p)
	case *varRef:
		return e.name == p.name
	}
	return false
}

// Identifiers that may not conflict:
//  - language keywords
//  - the contract name
//  - clause names
//  - contract params
//  - clause params
// However, two sibling clauses _may_ reuse the same parameter names (including "spends" identifiers).
func prohibitNameCollisions(contract *contract) error {
	topLevelNames := make(map[string]string) // maps identifiers to a description of their first use
	for _, k := range keywords {
		topLevelNames[k] = "keyword"
	}
	for _, b := range builtins {
		topLevelNames[b.name] = "built-in function"
	}
	if desc, ok := topLevelNames[contract.name]; ok {
		return fmt.Errorf("contract name \"%s\" conflicts with %s", contract.name, desc)
	}
	topLevelNames[contract.name] = "contract name"
	for _, p := range contract.params {
		if desc, ok := topLevelNames[p.name]; ok {
			return fmt.Errorf("contract parameter \"%s\" conflicts with %s", p.name, desc)
		}
		topLevelNames[p.name] = "contract parameter"
	}

	// clause names are top-level names
	for _, clause := range contract.clauses {
		if desc, ok := topLevelNames[clause.name]; ok {
			return fmt.Errorf("clause name \"%s\" conflicts with %s", clause.name, desc)
		}
		topLevelNames[clause.name] = "clause name"
	}

	// clause params are local to clauses
	for _, clause := range contract.clauses {
		clauseNames := make(map[string]string)
		for k, v := range topLevelNames {
			clauseNames[k] = v
		}
		for _, p := range clause.params {
			if desc, ok := clauseNames[p.name]; ok {
				return fmt.Errorf("parameter \"%s\" of clause \"%s\" conflicts with %s", p.name, clause.name, desc)
			}
			clauseNames[p.name] = fmt.Sprintf("clause parameter")
		}
	}

	return nil
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
			if referencedParam(stmt.expr) == p {
				count++
			}
		case *outputStatement:
			if len(stmt.call.args) == 1 && referencedParam(stmt.call.args[0]) == p {
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

func referencedParam(expr expression) *param {
	switch e := expr.(type) {
	case *varRef:
		return e.param
	case *propRef:
		return referencedParam(e.expr)
	}
	return nil
}

func referencedBuiltin(expr expression) *builtin {
	switch e := expr.(type) {
	case *varRef:
		return e.builtin

	case *propRef:
		t := typeOf(e)
		m := properties[t]
		if m != nil {
			if m[e.property] == "Function" {
				// xxx find the builtin
			}
		}
	}
	return nil
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

	case *varRef:
		for _, b := range builtins {
			if e.name == b.name {
				e.builtin = &b
				return nil
			}
		}
		for _, p := range contract.params {
			if e.name == p.name {
				e.param = p
				return nil
			}
		}
		for _, p := range clause.params {
			if e.name == p.name {
				e.param = p
				return nil
			}
		}
		return fmt.Errorf("undefined variable \"%s\" in clause \"%s\"", e.name, clause.name)

	case *propRef:
		return decorateRefsInExpr(contract, clause, e.expr)
	}
	return nil
}

func decorateOutputs(contract *contract, clause *clause) error {
	for _, s := range clause.statements {
		stmt, ok := s.(*outputStatement)
		if !ok {
			continue
		}
		if t := typeOf(stmt.call.fn); t != "Address" {
			return fmt.Errorf("type of function (%s) in output statement of clause \"%s\" is \"%s\", must be Address", stmt.call.fn, clause.name, t)
		}
		if len(stmt.call.args) != 1 {
			return fmt.Errorf("not yet supported: zero or multiple arguments in call to \"%s\" in output statement of clause \"%s\"", stmt.call.fn, clause.name)
		}
		if t := typeOf(stmt.call.args[0]); t != "Value" {
			return fmt.Errorf("not yet supported: argument of non-Value type \"%s\" passed to \"%s\" in output statement of clause \"%s\"", t, stmt.call.fn, clause.name)
		}
		p := referencedParam(stmt.call.args[0])
		if p == contract.params[len(contract.params)-1] {
			// The contract value param doesn't have to be matched against
			// an AssetAmount parameter.
			continue
		}

		// Look for a verify statement matching stmt.call.args[0] to an
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
					return referencedParam(prop.expr) == p && prop.property == "assetAmount"
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
			if typeOf(other) != "AssetAmount" {
				continue
			}
			v.associatedOutput = stmt
			stmt.param = referencedParam(other)
			if stmt.param == nil {
				return fmt.Errorf("cannot statically determine an AssetAmount parameter in \"%s\" in clause \"%s\" to check \"%s\" against", other, clause.name, p.name)
			}
			found = true
			break
		}
		if !found {
			return fmt.Errorf("Value parameter \"%s\" is in an output statement in clause \"%s\" but not checked in a verify statement", p.name, clause.name)
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

func typeCheckClause(contract *contract, clause *clause) error {
	for i, s := range clause.statements {
		switch stmt := s.(type) {
		case *verifyStatement:
			if stmt.associatedOutput != nil {
				// This verify is associated with an output. It doesn't get
				// compiled; instead it contributes its terms to the output
				// statement's CHECKOUTPUT.
				continue
			}
			if t := typeOf(stmt.expr); t != "Boolean" {
				return fmt.Errorf("expression in verify statement in clause \"%s\" has type \"%s\", must be Boolean", clause.name, t)
			}

		case *returnStatement:
			if i != len(clause.statements)-1 {
				return fmt.Errorf("return must be the final statement of clause \"%s\"", clause.name)
			}
			if t := typeOf(stmt.expr); t != "Value" {
				return fmt.Errorf("expression \"%s\" in return statement of clause \"%s\" has type \"%s\", must be Value", stmt.expr, clause.name, t)
			}
			if referencedParam(stmt.expr) != contract.params[len(contract.params)-1] {
				return fmt.Errorf("expression in return statement of clause \"%s\" must be the contract Value parameter", clause.name)
			}
		}
	}
	return nil
}

func typeCheckExpr(expr expression) error {
	switch e := expr.(type) {
	case *binaryExpr:
		lType := typeOf(e.left)
		rType := typeOf(e.right)

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
		if e.op.operand != "" && typeOf(e.expr) != e.op.operand {
			return fmt.Errorf("in \"%s\", operand has type \"%s\", must be \"%s\"", e, typeOf(e.expr), e.op.operand)
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
			if b.args[i] != "" && typeOf(actual) != b.args[i] {
				return fmt.Errorf("argument %d to \"%s\" has type \"%s\", must be \"%s\"", i, b.name, typeOf(actual), b.args[i])
			}
		}
	}
	return nil
}
