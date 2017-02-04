package filter

import (
	"errors"
	"fmt"

	chainerrors "chain/errors"
)

func TypeCheck(p Predicate, tbl *SQLTable, vals []interface{}) error {
	err := typeCheck(p.expr, tbl, vals)
	if err != nil {
		return chainerrors.WithDetail(ErrBadFilter, err.Error())
	}
	return nil
}

func isType(got Type, want Type) bool {
	return got == want || got == Any
}

func knownType(t Type) bool {
	return t == Bool || t == String || t == Integer || t == Object
}

func valueTypes(vals []interface{}) ([]Type, error) {
	valTypes := make([]Type, len(vals))
	for i, val := range vals {
		switch val.(type) {
		case int, uint, int32, uint32, int64, uint64:
			valTypes[i] = Integer
		case string:
			valTypes[i] = String
		case bool:
			valTypes[i] = Bool
		default:
			return nil, fmt.Errorf("unsupported value type %T", val)
		}
	}
	return valTypes, nil
}

func typeCheck(expr expr, tbl *SQLTable, vals []interface{}) error {
	valTypes, err := valueTypes(vals)
	if err != nil {
		return err
	}
	typ, err := typeCheckExpr(expr, tbl, valTypes)
	if err != nil {
		return err
	}
	if typ != Bool {
		return fmt.Errorf("filter predicate must evaluate to bool, got %s", typ)
	}
	return nil
}

func typeCheckExpr(expr expr, tbl *SQLTable, valTypes []Type) (typ Type, err error) {
	if expr == nil { // no expr is a valid, bool type
		return Bool, nil
	}

	switch e := expr.(type) {
	case parenExpr:
		return typeCheckExpr(e.inner, tbl, valTypes)
	case binaryExpr:
		leftTyp, err := typeCheckExpr(e.l, tbl, valTypes)
		if err != nil {
			return leftTyp, err
		}
		rightTyp, err := typeCheckExpr(e.r, tbl, valTypes)
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
		if len(valTypes) == 0 {
			return Any, nil
		}
		if e.num <= 0 || e.num > len(valTypes) {
			return typ, fmt.Errorf("unbound placeholder: $%d", e.num)
		}
		return valTypes[e.num-1], nil
	case attrExpr:
		col, ok := tbl.Columns[e.attr]
		if !ok {
			return typ, fmt.Errorf("invalid attribute: %s", e.attr)
		}
		return col.Type, nil
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
		typ, err = typeCheckExpr(e.objExpr, tbl, valTypes)
		if err != nil {
			return typ, err
		}
		if !isType(typ, Object) {
			return typ, errors.New("selector `.` can only be used on objects")
		}
		return Any, nil
	case envExpr:
		fk, ok := tbl.ForeignKeys[e.ident]
		if !ok {
			return typ, fmt.Errorf("invalid environment `%s`", e.ident)
		}
		typ, err = typeCheckExpr(e.expr, fk.Table, valTypes)
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
