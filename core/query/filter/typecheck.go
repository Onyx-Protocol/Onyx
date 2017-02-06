package filter

import (
	"errors"
	"fmt"
	"strings"
)

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
	selectorTypes := make(map[string]Type)
	typ, err := typeCheckExpr(expr, tbl, valTypes, selectorTypes)
	if err != nil {
		return err
	}
	if typ != Bool {
		return fmt.Errorf("filter predicate must evaluate to bool, got %s", typ)
	}
	return nil
}

func typeCheckExpr(expr expr, tbl *SQLTable, valTypes []Type, selectorTypes map[string]Type) (typ Type, err error) {
	if expr == nil { // no expr is a valid, bool type
		return Bool, nil
	}

	switch e := expr.(type) {
	case parenExpr:
		return typeCheckExpr(e.inner, tbl, valTypes, selectorTypes)
	case binaryExpr:
		leftTyp, err := typeCheckExpr(e.l, tbl, valTypes, selectorTypes)
		if err != nil {
			return leftTyp, err
		}
		rightTyp, err := typeCheckExpr(e.r, tbl, valTypes, selectorTypes)
		if err != nil {
			return rightTyp, err
		}

		// All our operands require left and right types to be equal. If
		// one of our types is known but the other is not, we need to
		// remember the type.
		if !knownType(leftTyp) && knownType(rightTyp) {
			err := assertType(e.l, rightTyp, selectorTypes)
			if err != nil {
				return leftTyp, err
			}
			leftTyp = rightTyp
		}
		if !knownType(rightTyp) && knownType(leftTyp) {
			err := assertType(e.r, leftTyp, selectorTypes)
			if err != nil {
				return leftTyp, err
			}
			rightTyp = leftTyp
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
		typ, err = typeCheckExpr(e.objExpr, tbl, valTypes, selectorTypes)
		if err != nil {
			return typ, err
		}
		if !isType(typ, Object) {
			return typ, errors.New("selector `.` can only be used on objects")
		}
		if !knownType(typ) {
			// We know e.objExpr is being used as an object, so record that
			// e.objExpr's json path must be an object.
			err = assertType(e.objExpr, Object, selectorTypes)
			if err != nil {
				return typ, err
			}
		}

		// Unfortunately, we can't know the type of the field within the
		// object yet. Depending on the context, we might be able to assign it
		// a type later in assertType.
		return Any, nil
	case envExpr:
		fk, ok := tbl.ForeignKeys[e.ident]
		if !ok {
			return typ, fmt.Errorf("invalid environment `%s`", e.ident)
		}
		typ, err = typeCheckExpr(e.expr, fk.Table, valTypes, selectorTypes)
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

func assertType(expr expr, typ Type, selectorTypes map[string]Type) error {
	switch e := expr.(type) {
	case parenExpr:
		return assertType(e.inner, typ, selectorTypes)
	case placeholderExpr:
		// This is a special case for when we parse a txfeed filter at
		// txfeed creation time. We don't have access to concrete values
		// yet, so the parameters are untyped.
		return nil
	case selectorExpr:
		path := strings.Join(jsonbPath(expr), ".")
		boundTyp, ok := selectorTypes[path]
		if ok && boundTyp != typ {
			return fmt.Errorf("%q used as both %s and %s", path, boundTyp, typ)
		}
		selectorTypes[path] = typ
		return nil
	default:
		// This should be impossible because all other expressions are
		// strongly typed.
		panic(fmt.Errorf("unexpected assertType on %T", expr))
	}
}
