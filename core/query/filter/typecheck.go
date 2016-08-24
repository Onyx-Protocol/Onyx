package filter

import (
	"errors"
	"fmt"
)

func isType(got Type, want Type) bool {
	return got == want || got == Any
}

func knownType(t Type) bool {
	return t == Bool || t == String || t == Integer || t == Object
}

func typeCheck(expr expr) error {
	typ, err := typeCheckExpr(expr)
	if err != nil {
		return err
	}
	if typ != Bool {
		return fmt.Errorf("filter predicate must evaluate to bool, got %s", typ)
	}
	return nil
}

func typeCheckExpr(expr expr) (typ Type, err error) {
	if expr == nil { // no expr is a valid, bool type
		return Bool, nil
	}

	switch e := expr.(type) {
	case parenExpr:
		return typeCheckExpr(e.inner)
	case binaryExpr:
		leftTyp, err := typeCheckExpr(e.l)
		if err != nil {
			return leftTyp, err
		}
		rightTyp, err := typeCheckExpr(e.r)
		if err != nil {
			return rightTyp, err
		}

		switch e.op.name {
		case "OR", "AND":
			if !isType(leftTyp, Bool) || !isType(rightTyp, Bool) {
				return typ, fmt.Errorf("%s expects bool operands", e.op.name)
			}
			return Bool, nil
		case "=":
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
		default:
			panic(fmt.Errorf("unsupported operator: %s", e.op.name))
		}
	case placeholderExpr:
		return Any, nil
	case attrExpr:
		return Any, nil
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
		typ, err = typeCheckExpr(e.objExpr)
		if err != nil {
			return typ, err
		}
		if !isType(typ, Object) {
			return typ, errors.New("selector `.` can only be used on objects")
		}
		return Any, nil
	case envExpr:
		typ, err = typeCheckExpr(e.expr)
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
