package filter

import "strconv"

type binaryOp struct {
	precedence int
	name       string // AND, =, etc.
	apply      func(lv, rv value) value
}

var binaryOps = map[string]*binaryOp{
	"OR":  {1, "OR", applyOr},
	"AND": {2, "AND", applyAnd},
	"=":   {3, "=", applyEqual},
}

func applyOr(lv, rv value) value {
	// non-bool operands will have empty sets
	return value{t: Bool, set: union(lv.set, rv.set)}
}

func applyAnd(lv, rv value) value {
	// non-bool operands will have empty sets
	return value{t: Bool, set: intersection(lv.set, rv.set)}
}

func applyEqual(lv, rv value) value {
	var set Set
	switch {
	// static, known-value cases
	case lv.is(Bool) && rv.is(Bool):
		// (A ∩ B) ∪ (A ∪ B)´
		set = union(
			intersection(lv.set, rv.set),
			complement(union(lv.set, rv.set)),
		)
	case lv.is(Integer) && rv.is(Integer):
		set = Set{Invert: lv.integer == rv.integer}
	case lv.is(String) && rv.is(String):
		set = Set{Invert: lv.str == rv.str}

	// dynamic, placeholder cases
	case lv.is(Any) && rv.is(String):
		set = Set{Values: []string{rv.str}}
	case lv.is(String) && rv.is(Any):
		set = Set{Values: []string{lv.str}}
	case lv.is(Any) && rv.is(Integer):
		set = Set{Values: []string{strconv.Itoa(rv.integer)}}
	case lv.is(Integer) && rv.is(Any):
		set = Set{Values: []string{strconv.Itoa(lv.integer)}}

	// error cases
	case lv.is(Any) && rv.is(Any):
		panic("placeholders cannot be compared")
	case lv.is(Object) && rv.is(Object):
		set = Set{} // objects are never equal
	default:
		set = Set{} // different types are never equal
	}
	return value{t: Bool, set: set}
}
