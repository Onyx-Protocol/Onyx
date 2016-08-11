package chql

import (
	"bytes"
	"encoding/json"
	"strconv"
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

type SQLExpr struct {
	SQL     string
	Values  []interface{}
	GroupBy [][]string
}

func asSQL(e expr, dataColumn string, values []interface{}) (exp SQLExpr, err error) {
	pvals := map[int]interface{}{}
	for i, v := range values {
		pvals[i+1] = v
	}

	matches, bindings := matchingObjects(e, pvals)

	var buf bytes.Buffer
	var params []interface{}
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

	exp = SQLExpr{
		SQL:    buf.String(),
		Values: params,
	}
	for _, b := range bindings {
		exp.GroupBy = append(exp.GroupBy, b.path)
	}
	return exp, nil
}
