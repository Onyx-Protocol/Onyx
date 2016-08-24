package chql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/lib/pq"
)

// AsSQL translates q to SQL.
func AsSQL(q Query, dataColumn string, values []interface{}) (sqlExpr SQLExpr, err error) {
	defer func() {
		r := recover()
		if e, ok := r.(error); ok {
			err = e
		} else if r != nil {
			panic(r)
		}
	}()

	return asSQL(q.expr, dataColumn, values)
}

// FieldAsSQL returns a jsonb indexing SQL representation of the field.
func FieldAsSQL(col string, f Field) string {
	components := jsonbPath(f)

	var buf bytes.Buffer
	buf.WriteString(pq.QuoteIdentifier(col))
	for i, c := range components {
		if i+1 < len(components) {
			buf.WriteString("->")
		} else {
			buf.WriteString("->>")
		}

		// Note, field here originally came from an identifier in ChQL, so
		// it should be safe to embed in a string without quoting.
		// TODO(jackson): Quote/restrict anyways to be defensive.
		buf.WriteString("'")
		buf.WriteString(c)
		buf.WriteString("'")
	}
	return buf.String()
}

func jsonbPath(f Field) []string {
	switch e := f.expr.(type) {
	case selectorExpr:
		return append(jsonbPath(Field{expr: e.objExpr}), e.ident)
	case attrExpr:
		return []string{e.attr}
	default:
		panic(fmt.Errorf("unexpected field of type %T", e))
	}
}

type SQLExpr struct {
	SQL    string
	Values []interface{}
}

func asSQL(e expr, dataColumn string, values []interface{}) (exp SQLExpr, err error) {
	if e == nil {
		// An empty expression is a valid query without any filtering.
		return SQLExpr{}, nil
	}

	pvals := map[int]interface{}{}
	for i, v := range values {
		if v != nil {
			pvals[i+1] = v
		}
	}

	matches := matchingObjects(e, pvals)

	var buf bytes.Buffer
	var params []interface{}
	if len(matches) > 1 {
		buf.WriteString("(")
	}
	for i, condition := range matches {
		if i > 0 {
			buf.WriteString(" OR ")
		}

		b, err := json.Marshal(condition)
		if err != nil {
			return exp, err
		}

		params = append(params, string(b))
		buf.WriteString("(" + dataColumn + " @> $" + strconv.Itoa(len(params)) + "::jsonb)")
	}
	if len(matches) > 1 {
		buf.WriteString(")")
	}

	return SQLExpr{
		SQL:    buf.String(),
		Values: params,
	}, nil
}
