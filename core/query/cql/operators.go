package cql

import "strconv"

type binaryOp struct {
	precedence int
	name       string // AND, <=, etc.
	apply      func(lv, rv value) value
}

var binaryOps = map[string]*binaryOp{
	"OR":       {1, "OR", applyOr},
	"AND":      {2, "AND", applyAnd},
	"<":        {3, "<", applyLessThan},
	">":        {3, ">", applyGreaterThan},
	"<=":       {3, "<=", applyLessThanEqual},
	">=":       {3, ">=", applyGreaterThanEqual},
	"=":        {3, "=", applyEqual},
	"!=":       {3, "!=", applyNotEqual},
	"CONTAINS": {4, "CONTAINS", applyContains},
}

func applyOr(lv, rv value) value {
	if !lv.is(boolTyp) || !rv.is(boolTyp) {
		panic("OR requires boolean operands")
	}
	return value{t: boolTyp, set: union(lv.set, rv.set)}
}

func applyAnd(lv, rv value) value {
	if !lv.is(boolTyp) || !rv.is(boolTyp) {
		panic("AND requires boolean operands")
	}
	return value{t: boolTyp, set: intersection(lv.set, rv.set)}
}

func applyLessThan(lv, rv value) value {
	if lv.is(placeholderTyp) || rv.is(placeholderTyp) {
		panic("inequality comparison does not support parameterized expressions")
	}
	if lv.is(integerTyp) && rv.is(integerTyp) {
		return value{t: boolTyp, set: Set{Invert: lv.integer < rv.integer}}
	}
	if lv.is(stringTyp) && rv.is(stringTyp) {
		return value{t: boolTyp, set: Set{Invert: lv.str < rv.str}}
	}
	panic("inequality comparison requires scalar operands")
}

func applyLessThanEqual(lv, rv value) value {
	if lv.is(placeholderTyp) || rv.is(placeholderTyp) {
		panic("inequality comparison does not support parameterized expressions")
	}
	if lv.is(integerTyp) && rv.is(integerTyp) {
		return value{t: boolTyp, set: Set{Invert: lv.integer <= rv.integer}}
	}
	if lv.is(stringTyp) && rv.is(stringTyp) {
		return value{t: boolTyp, set: Set{Invert: lv.str <= rv.str}}
	}
	panic("inequality comparison requires scalar operands")
}

func applyGreaterThan(lv, rv value) value {
	v := applyLessThanEqual(lv, rv)
	v.set.Invert = !v.set.Invert
	return v
}

func applyGreaterThanEqual(lv, rv value) value {
	v := applyLessThan(lv, rv)
	v.set.Invert = !v.set.Invert
	return v
}

func applyEqual(lv, rv value) value {
	var set Set
	switch {
	// static, known-value cases
	case lv.is(boolTyp) && rv.is(boolTyp):
		// (A ∩ B) ∪ (A ∪ B)´
		set = union(
			intersection(lv.set, rv.set),
			complement(union(lv.set, rv.set)),
		)
	case lv.is(integerTyp) && rv.is(integerTyp):
		set = Set{Invert: lv.integer == rv.integer}
	case lv.is(stringTyp) && rv.is(stringTyp):
		set = Set{Invert: lv.str == rv.str}

	// dynamic, placeholder cases
	case lv.is(placeholderTyp) && rv.is(stringTyp):
		set = Set{Values: []string{rv.str}}
	case lv.is(stringTyp) && rv.is(placeholderTyp):
		set = Set{Values: []string{lv.str}}
	case lv.is(placeholderTyp) && rv.is(integerTyp):
		set = Set{Values: []string{strconv.Itoa(rv.integer)}}
	case lv.is(integerTyp) && rv.is(placeholderTyp):
		set = Set{Values: []string{strconv.Itoa(lv.integer)}}

	// error cases
	case lv.is(placeholderTyp) && rv.is(placeholderTyp):
		panic("placeholders cannot be compared")
	case lv.is(listTyp) && rv.is(listTyp):
		panic("lists cannot be compared with comparison operators")
	default:
		panic("mismatched types for comparison operator")
	}
	return value{t: boolTyp, set: set}
}

func applyNotEqual(lv, rv value) value {
	neq := applyEqual(lv, rv)
	neq.set.Invert = !neq.set.Invert
	return neq
}

func applyContains(lv, rv value) value {
	if !lv.is(listTyp) {
		panic("CONTAINS requires left operand to be list")
	}
	if rv.is(placeholderTyp) {
		return value{t: boolTyp, set: Set{Values: lv.list}}
	}
	if !rv.is(stringTyp) {
		panic("CONTAINS requires right operand to be string")
	}

	found := false
	for _, v := range lv.list {
		found = found || v == rv.str
	}
	return value{t: boolTyp, set: Set{Invert: found}}
}
