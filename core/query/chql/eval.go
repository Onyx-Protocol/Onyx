package chql

import (
	"fmt"
	"sort"
	"strconv"
)

// Eval evaluates the provided query against the provided environment.
func Eval(env map[string]interface{}, q Query) (s Set, err error) {
	if q.Parameters > 1 {
		return s, fmt.Errorf("multiple parameters are not yet supported")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("preparing query: %s", r)
		}
	}()

	v := eval(mapEnv(env), q.expr)
	if !v.is(Bool) {
		return s, fmt.Errorf("query `%s` does not evaluate to a boolean", q.expr.String())
	}

	// Always sort the return set so Eval(...) returns a
	// consistent, canonical representation of a set.
	sort.Strings(s.Values)
	return v.set, nil
}

// Type defines the value types of ChQL.
type Type int

const (
	Any Type = iota
	Bool
	String
	Integer
	Object
)

func (t Type) String() string {
	switch t {
	case Any:
		return "any"
	case Bool:
		return "bool"
	case String:
		return "string"
	case Integer:
		return "integer"
	case Object:
		return "object"
	}
	panic("unknown ChQL type")
}

// value represents the result of evaluating an expression against an
// environment (a transaction, an unspent output, etc).
//
// Boolean values are represented as the set of parameter values that
// satisfy the expression:
// - true is represented as the complement of the empty set because
//   all parameter values satisfy the expression.
// - false is represented as the empty set because there are no possible
//   parameter values that could satisfy the expression.
type value struct {
	t       Type
	set     Set                    // Bool
	str     string                 // String
	integer int                    // Integer
	obj     map[string]interface{} // Object
}

func (v value) is(t Type) bool {
	return v.t == t
}

func eval(env environment, expr expr) value {
	switch e := expr.(type) {
	case binaryExpr:
		lv, rv := eval(env, e.l), eval(env, e.r)
		return e.op.apply(lv, rv)
	case attrExpr:
		return env.Get(e.attr)
	case parenExpr:
		return eval(env, e.inner)
	case valueExpr:
		switch e.typ {
		case tokString:
			return value{t: String, str: e.value[1 : len(e.value)-1]}
		case tokInteger:
			v, err := strconv.Atoi(e.value)
			if err != nil {
				panic(err) // can't happen; caught during parsing
			}
			return value{t: Integer, integer: v}
		default:
			panic(fmt.Errorf("value expr with invalid token type: %s", e.typ))
		}
	case envExpr:
		subenvs := env.Environments(e.ident)
		var set Set
		for _, subenv := range subenvs {
			v := eval(subenv, e.expr)
			if !v.is(Bool) {
				// type error; return false
				return value{t: Bool, set: Set{}}
			}
			set = union(set, v.set)
		}
		return value{t: Bool, set: set}
	case selectorExpr:
		v := eval(env, e.objExpr)
		if !v.is(Object) {
			// type error; return false
			return value{t: Bool, set: Set{}}
		}

		m := mapEnv(v.obj)
		return m.Get(e.ident)
	case placeholderExpr:
		return value{t: Any}
	default:
		panic(fmt.Errorf("unrecognized expr type %T", expr))
	}
}
