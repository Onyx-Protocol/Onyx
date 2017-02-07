package filter

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/lib/pq"

	"chain/errors"
)

// AsSQL translates p to SQL.
func AsSQL(p Predicate, tbl *SQLTable, values []interface{}) (q string, err error) {
	defer func() {
		r := recover()
		if e, ok := r.(error); ok {
			err = e
		} else if r != nil {
			panic(r)
		}
	}()

	b := &sqlBuilder{
		values:        values,
		baseTbl:       tbl,
		selectorTypes: p.selectorTypes,
	}
	c := &sqlContext{sqlBuilder: b, tbl: tbl}
	err = asSQL(c, p.expr)
	if err != nil {
		return "", err
	}
	return b.buf.String(), nil
}

// FieldAsSQL returns a SQL representation of the field.
func FieldAsSQL(tbl *SQLTable, f Field) (string, error) {
	path := jsonbPath(f.expr)

	base, rest := path[0], path[1:]
	col, ok := tbl.Columns[base]
	if !ok {
		return "", errors.WithDetailf(ErrBadFilter, "invalid attribute: %s", base)
	}
	if col.SQLType != SQLJSONB && len(rest) > 0 {
		return "", errors.WithDetailf(ErrBadFilter, "cannot index on non-object attribute: %s", base)
	}

	var buf bytes.Buffer
	if col.SQLType == SQLBytea {
		buf.WriteString("encode(")
	}
	buf.WriteString(tbl.Alias)
	buf.WriteRune('.')
	buf.WriteString(pq.QuoteIdentifier(base))
	if col.SQLType == SQLBytea {
		buf.WriteString(", 'hex')")
	}

	for i, c := range rest {
		if i == len(rest)-1 {
			buf.WriteString("->>")
		} else {
			buf.WriteString("->")
		}

		// Note, field here originally came from an identifier in a filter, so
		// it should be safe to embed in a string without quoting.
		// TODO(jackson): Quote/restrict anyways to be defensive.
		buf.WriteString("'")
		buf.WriteString(c)
		buf.WriteString("'")
	}
	return buf.String(), nil
}

func jsonbPath(f expr) []string {
	switch e := f.(type) {
	case selectorExpr:
		return append(jsonbPath(e.objExpr), e.ident)
	case attrExpr:
		return []string{e.attr}
	default:
		panic(fmt.Errorf("unexpected field of type %T", e))
	}
}

type SQLType int

const (
	SQLBool SQLType = iota
	SQLText
	SQLBytea
	SQLJSONB
	SQLInteger
	SQLBigint
	SQLTimestamp
)

type SQLTable struct {
	Name        string
	Alias       string
	Columns     map[string]*SQLColumn
	ForeignKeys map[string]*SQLForeignKey
}

type SQLColumn struct {
	Name    string
	Type    Type
	SQLType SQLType
}

type SQLForeignKey struct {
	Table         *SQLTable
	LocalColumn   string
	ForeignColumn string
}

type sqlBuilder struct {
	baseTbl       *SQLTable
	values        []interface{}
	selectorTypes map[string]Type
	buf           bytes.Buffer
}

type sqlContext struct {
	*sqlBuilder
	tbl *SQLTable
}

func (c *sqlContext) writeCol(name string) {
	c.buf.WriteString(c.tbl.Alias)
	c.buf.WriteRune('.')
	c.buf.WriteString(pq.QuoteIdentifier(name))
}

func asSQL(c *sqlContext, filterExpr expr) error {
	switch e := filterExpr.(type) {
	case parenExpr:
		c.buf.WriteRune('(')
		err := asSQL(c, e.inner)
		if err != nil {
			return err
		}
		c.buf.WriteRune(')')
	case valueExpr:
		switch e.typ {
		case tokString:
			c.buf.WriteString(e.value)
		case tokInteger:
			c.buf.WriteString(e.value)
			c.buf.WriteString(`::bigint`)
		default:
			return errors.WithDetailf(ErrBadFilter, "value expr with invalid token type: %s", e.typ)
		}
	case attrExpr:
		col, ok := c.tbl.Columns[e.attr]
		if !ok {
			return errors.WithDetailf(ErrBadFilter, "invalid attribute: %s", e.attr)
		}

		// How we select the column in SQL depends on the column type.
		switch col.SQLType {
		case SQLBytea:
			c.buf.WriteString("encode(")
			c.writeCol(col.Name)
			c.buf.WriteString(`, 'hex')`)
		case SQLJSONB:
			c.writeCol(col.Name)
			c.buf.WriteString(`::text`)
		case SQLInteger:
			c.writeCol(col.Name)
			c.buf.WriteString(`::bigint`)
		case SQLTimestamp:
			c.writeCol(col.Name)
			c.buf.WriteString(`::text`)
		case SQLBool, SQLText, SQLBigint:
			c.writeCol(col.Name)
		default:
			panic(fmt.Errorf("unknown sql type: %d", col.SQLType))
		}
	case selectorExpr:
		// unwind the jsonb path
		path := jsonbPath(e)
		selectorPath := strings.Join(path, ".")
		base, path := path[0], path[1:]

		col, ok := c.tbl.Columns[base]
		if !ok {
			return errors.WithDetailf(ErrBadFilter, "invalid attribute: %s", base)
		}
		if col.SQLType != SQLJSONB {
			return errors.WithDetailf(ErrBadFilter, "cannot index on non-object attribute: %s", base)
		}

		c.buf.WriteRune('(')
		c.writeCol(base)
		for i, p := range path {
			if i == len(path)-1 {
				c.buf.WriteString(`->>`)
			} else {
				c.buf.WriteString(`->`)
			}
			c.buf.WriteRune('\'')
			c.buf.WriteString(p)
			c.buf.WriteRune('\'')
		}
		c.buf.WriteRune(')')

		// Use the type inferred by the typechecker to cast the expression
		// o the right type. If uncasted, the ->> operator will result in a
		// text PostgreSQL value.
		typ, ok := c.selectorTypes[selectorPath]
		if ok {
			switch typ {
			case Integer:
				c.buf.WriteString("::bigint")
			case Bool:
				c.buf.WriteString("::boolean")
			case Object:
				c.buf.WriteString("::jsonb")
			case Any, String:
				// don't do anything (defaulting to text)
			default:
				panic(fmt.Errorf("unknown type %s", typ))
			}
		}
	case binaryExpr:
		err := asSQL(c, e.l)
		if err != nil {
			return err
		}

		c.buf.WriteRune(' ')
		c.buf.WriteString(e.op.sqlOp)
		c.buf.WriteRune(' ')

		err = asSQL(c, e.r)
		if err != nil {
			return err
		}
	case placeholderExpr:
		if e.num < 1 || e.num > len(c.values) {
			return errors.WithDetailf(ErrBadFilter, "unbound placeholder: $%d", e.num)
		}
		c.buf.WriteRune('$')
		c.buf.WriteString(strconv.Itoa(e.num))
	case envExpr:
		fk, ok := c.tbl.ForeignKeys[e.ident]
		if !ok {
			return errors.WithDetailf(ErrBadFilter, "invalid environment `%s`", e.ident)
		}

		// Create a new sql context for the join and convert it to sql too.
		subCtx := &sqlContext{
			sqlBuilder: c.sqlBuilder,
			tbl:        fk.Table,
		}
		subCtx.buf.WriteRune('\n')
		subCtx.buf.WriteString(`EXISTS(SELECT 1 FROM `)
		subCtx.buf.WriteString(fk.Table.Name)
		subCtx.buf.WriteString(` AS `)
		subCtx.buf.WriteString(fk.Table.Alias)
		subCtx.buf.WriteString(` WHERE `)
		subCtx.buf.WriteString(fk.Table.Alias)
		subCtx.buf.WriteString(`."`)
		subCtx.buf.WriteString(fk.ForeignColumn)
		subCtx.buf.WriteString(`" = `)
		subCtx.buf.WriteString(c.tbl.Alias)
		subCtx.buf.WriteString(`."`)
		subCtx.buf.WriteString(fk.LocalColumn)
		subCtx.buf.WriteString(`" AND (`)

		// Serialize the environment's expression.
		err := asSQL(subCtx, e.expr)
		if err != nil {
			return err
		}

		subCtx.buf.WriteString(`))`)
		subCtx.buf.WriteRune('\n')
	}
	return nil
}
