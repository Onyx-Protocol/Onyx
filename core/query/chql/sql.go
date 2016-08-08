package chql

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/lib/pq"
)

// AsSQL translates q to SQL using the provided sql table definition.
//
// Example usage:
//     sqlExpr, err := AsSQL(chqlQuery, unspentOutputsTbl)
//     if err != nil {
//         // ...
//     }
//     const q = `SELECT ... FROM unspent_outputs WHERE `
//     rows, err := db.Query(q + sqlExpr.String(), sqlExpr.Values(x, y, z)...)
//     // ...
//
func AsSQL(q Query, tbl SQLTable) (sqlExpr SQLExpr, err error) {
	defer func() {
		r := recover()
		if e, ok := r.(error); ok {
			err = e
		} else if r != nil {
			panic(r)
		}
	}()

	// Type check the query with the type annotations from tbl.
	err = typeCheck(q.expr, tbl)
	if err != nil {
		return sqlExpr, err
	}

	// Translate to SQL, writing the query into sqlExpr's buffer.
	translateToSQL(&sqlExpr, tbl, q.expr)
	return sqlExpr, err
}

// SQLTable defines the schema of a queryable SQL table and the mapping
// of ChQL attribute names to column names.
type SQLTable map[string]SQLColumn

// SQLColumn defines a column of a queryable SQL table.
type SQLColumn struct {
	Name string
	Type Type
}

type sqlPlaceholder struct {
	number int
	value  interface{}
}

// SQLExpr encapsulates a generated SQL expression for executing a ChQL query
// against a table. The SQL expression is guaranteed to evaluate to a boolean
// in SQL against the provided table.
type SQLExpr struct {
	buf              bytes.Buffer
	placeholderCount int
	placeholders     []sqlPlaceholder
}

// String returns the SQL representation of the expression.
func (q SQLExpr) String() string {
	return q.buf.String()
}

// Values constructs the values to the parameterized SQL query by merging
// in the ChQL query parameters.
func (q SQLExpr) Values(params ...interface{}) []interface{} {
	var values []interface{}
	for _, p := range q.placeholders {
		if p.value != nil {
			values = append(values, p.value)
			continue
		}
		values = append(values, params[p.number-1])
	}
	return values
}

func translateToSQL(w *SQLExpr, t SQLTable, expr expr) {
	switch e := expr.(type) {
	case parenExpr:
		// No need to write extra parens here because all expressions
		// with operators are written with explicit wrapping parens so
		// as not to assume SQL operator precedence is the same as ChQL's.
		translateToSQL(w, t, e.inner)
	case notExpr:
		w.buf.WriteString("(")
		w.buf.WriteString("NOT ")
		translateToSQL(w, t, e.inner)
		w.buf.WriteString(")")
	case binaryExpr:
		// translate the left operand
		w.buf.WriteString("(")
		translateToSQL(w, t, e.l)

		// translate the operator itself
		w.buf.WriteString(" ")
		switch e.op.name {
		case "OR", "AND", "<", "<=", ">", ">=", "=", "!=":
			w.buf.WriteString(e.op.name)
		default:
			panic(fmt.Errorf("unsupported operator: %s", e.op.name))
		}
		w.buf.WriteString(" ")

		// translate the right operand
		translateToSQL(w, t, e.r)
		w.buf.WriteString(")")
	case placeholderExpr:
		w.placeholderCount++
		w.placeholders = append(w.placeholders, sqlPlaceholder{number: e.num})
		w.buf.WriteString("$")
		w.buf.WriteString(strconv.Itoa(w.placeholderCount))
	case attrExpr:
		column, ok := t[e.attr]
		if !ok {
			panic(fmt.Errorf("unknown column %q", e.attr))
		}
		w.buf.WriteString(pq.QuoteIdentifier(column.Name))
	case valueExpr:
		switch e.typ {
		case tokString:
			v := e.value[1 : len(e.value)-1]
			w.placeholderCount++
			w.placeholders = append(w.placeholders, sqlPlaceholder{value: v})

			w.buf.WriteString("$")
			w.buf.WriteString(strconv.Itoa(w.placeholderCount))
		case tokInteger:
			w.buf.WriteString(e.value)
		default:
			panic(fmt.Errorf("value expr with invalid token type: %s", e.typ))
		}
	case selectorExpr:
		// TODO(jackson): implement
		panic("not yet implemented")
	case envExpr:
		panic("environment expressions unsupported")
	default:
		panic(fmt.Errorf("unrecognized expr type %T", expr))
	}
}
