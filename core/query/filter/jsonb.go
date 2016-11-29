package filter

import (
	"fmt"
	"strconv"

	"chain-stealth/errors"
)

func jsonValue(expr expr, pvals map[int]interface{}) (v interface{}, path []string) {
	switch e := expr.(type) {
	case parenExpr:
		return jsonValue(expr, pvals)
	case placeholderExpr:
		return pvals[e.num], nil
	case attrExpr:
		// TODO(jackson): Handle implicit booleans like `inputs(is_issuance)`
		return nil, []string{e.attr}
	case selectorExpr:
		_, innerPath := jsonValue(e.objExpr, pvals)
		return nil, append([]string{e.ident}, innerPath...)
	case valueExpr:
		if e.typ == tokString {
			strv := e.value[1 : len(e.value)-1]
			return strv, nil
		}
		if e.typ == tokInteger {
			i, _ := strconv.Atoi(e.value) // err impossible; enforce at parser
			return i, nil
		}
		panic(fmt.Errorf("value expr with invalid token type: %s", e.typ))
	default:
		panic(fmt.Errorf("unexpected expr %T", expr))
	}
}

func matchingObjects(expr expr, pvals map[int]interface{}) []interface{} {
	switch e := expr.(type) {
	case parenExpr:
		return matchingObjects(e.inner, pvals)
	case envExpr:
		conds := matchingObjects(e.expr, pvals)
		var newConditions []interface{}
		for _, v := range conds {
			newConditions = append(newConditions, map[string]interface{}{
				e.ident: []interface{}{v},
			})
		}
		return newConditions
	case binaryExpr:
		if e.op.name == "OR" {
			return append(matchingObjects(e.l, pvals), matchingObjects(e.r, pvals)...)
		}

		if e.op.name == "AND" {
			// TODO: restrict the complexity of queries to prevent people
			// from shooting themselves in the foot with an enormous
			// cross product.
			leftConds := matchingObjects(e.l, pvals)
			rightConds := matchingObjects(e.r, pvals)
			var intersection []interface{}
			for _, c1 := range leftConds {
				for _, c2 := range rightConds {
					intersection = append(intersection, mergeObjects(c1, c2))
				}
			}
			return intersection
		}

		if e.op.name == "=" {
			lv, lp := jsonValue(e.l, pvals)
			rv, rp := jsonValue(e.r, pvals)
			switch {
			// left is a value, right is a path
			case lv != nil && len(rp) > 0:
				m := lv
				for _, p := range rp {
					m = map[string]interface{}{p: m}
				}
				return []interface{}{m}

			// right is a value, left is a path
			case rv != nil && len(lp) > 0:
				m := rv
				for _, p := range lp {
					m = map[string]interface{}{p: m}
				}
				return []interface{}{m}

			default:
				panic(errors.WithDetail(ErrBadFilter, "unsupported operands for ="))
			}
		}
		panic(fmt.Errorf("unknown operator %q", e.op.name))
	}
	panic(fmt.Errorf("unexpected expr type %T", expr))
}

func mergeObjects(o1, o2 interface{}) interface{} {
	s1, ok1 := o1.([]interface{})
	s2, ok2 := o2.([]interface{})
	if ok1 && ok2 {
		var combined []interface{}
		combined = append(combined, s1...)
		combined = append(combined, s2...)
		return combined
	}

	m1, ok1 := o1.(map[string]interface{})
	m2, ok2 := o2.(map[string]interface{})
	if !ok1 || !ok2 {
		panic(fmt.Errorf("expect map[string]interface{} got %T and %T", o1, o2))
	}

	m := map[string]interface{}{}
	sharedKeys := map[string]bool{}
	for k, v := range m1 {
		if _, ok := m2[k]; ok {
			sharedKeys[k] = true
		} else {
			m[k] = v
		}
	}
	for k, v := range m2 {
		if _, ok := m1[k]; ok {
			sharedKeys[k] = true
		} else {
			m[k] = v
		}
	}
	for sharedKey := range sharedKeys {
		m[sharedKey] = mergeObjects(m1[sharedKey], m2[sharedKey])
	}
	return m
}
