package cql

import (
	"errors"
	"fmt"
)

func isType(got Type, want Type) bool {
	return got == want || got == Any
}

func knownType(t Type) bool {
	return t == Bool || t == String || t == Integer || t == List || t == Object
}

func typeCheck(expr expr, t SQLTable) error {
	typ, err := typeCheckExpr(expr, t)
	if err != nil {
		return err
	}
	if typ != Bool {
		return fmt.Errorf("query must evaluate to bool, got %s", typ)
	}
	return nil
}

func typeCheckExpr(expr expr, t SQLTable) (typ Type, err error) {
	switch e := expr.(type) {
	case parenExpr:
		return typeCheckExpr(e.inner, t)
	case notExpr:
		typ, err = typeCheckExpr(e.inner, t)
		if err != nil {
			return typ, err
		}
		if !isType(typ, Bool) {
			return typ, fmt.Errorf("NOT expects a bool operand")
		}
		return Bool, nil
	case binaryExpr:
		leftTyp, err := typeCheckExpr(e.l, t)
		if err != nil {
			return leftTyp, err
		}
		rightTyp, err := typeCheckExpr(e.r, t)
		if err != nil {
			return rightTyp, err
		}

		switch e.op.name {
		case "OR", "AND":
			if !isType(leftTyp, Bool) || !isType(rightTyp, Bool) {
				return typ, fmt.Errorf("%s expects bool operands", e.op.name)
			}
			return Bool, nil
		case "<", "<=", ">", ">=", "=", "!=":
			if !isType(leftTyp, String) && !isType(leftTyp, Integer) {
				return typ, fmt.Errorf("%s expects integer or string operands", e.op.name)
			}
			if !isType(rightTyp, String) && !isType(rightTyp, Integer) {
				return typ, fmt.Errorf("%s expects integer or string operands", e.op.name)
			}
			if knownType(rightTyp) && knownType(leftTyp) && leftTyp != rightTyp {
				return typ, fmt.Errorf("%s expects operands of matching types", e.op.name)
			}
			return Bool, nil
		case "CONTAINS":
			if !isType(leftTyp, List) {
				return typ, fmt.Errorf("CONTAINS expects left operand to be a list")
			}
			if !isType(rightTyp, String) {
				return typ, fmt.Errorf("CONTAINS expects right operand to be a string")
			}
			return Bool, nil
		default:
			panic(fmt.Errorf("unsupported operator: %s", e.op.name))
		}
	case placeholderExpr:
		return Any, nil
	case attrExpr:
		if t == nil {
			return Any, nil
		}

		column, ok := t[e.attr]
		if !ok {
			return typ, fmt.Errorf("unknown column %q", e.attr)
		}
		return column.Type, nil
	case valueExpr:
		switch e.typ {
		case tokString:
			return String, nil
		case tokInteger:
			return Integer, nil
		default:
			panic(fmt.Errorf("value expr with invalid token type: %s", e.typ))
		}
	case selectorExpr:
		typ, err = typeCheckExpr(e.objExpr, t)
		if err != nil {
			return typ, err
		}
		if !isType(typ, Object) {
			return typ, errors.New("selector `.` can only be used on objects")
		}
		return Any, nil
	case envExpr:
		typ, err = typeCheckExpr(e.expr, t)
		if err != nil {
			return typ, err
		}
		if typ != Bool {
			return typ, errors.New(e.ident + "(...) body must have type bool")
		}
		return Bool, nil
	default:
		panic(fmt.Errorf("unrecognized expr type %T", expr))
	}
}
