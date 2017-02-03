/*
Package filter parses and evaluates Chain filter expressions. A
predicate is a boolean expression with zero or more placeholder
values ($1, $2, etc) that are initially unconstrained. The
predicate is evaluated in an environment (such as a transaction
object or a UTXO) that determines the value of all non-placeholder
terms. The predicate and its fixed values together constrain the
placeholders.

Expressions in a filter expression have the following forms:

  Form                     Type     Subexpression types
  expr1 "OR" expr2         bool     bool, bool
  expr1 "AND" expr2        bool     bool, bool
  ident "(" expr ")"       bool     list, bool
  expr1 "=" expr2          bool     any (must match)
  expr "." ident           any      object
  "(" expr ")"             any      any
  ident                    any      n/a
  placeholder              scalar   n/a
  string                   string   n/a
  int                      int      n/a

  ident is an alphanumeric identifier
  placeholder is a decimal int with prefix "$"
  scalar means int or string
  string is single-quoted, and cannot contain backslash
  int is decimal or hexadecimal (with prefix "0x")
  list is a slice of environments

The environment is a map from names to values. Identifier
expressions get their values from the environment map.

The form 'ident(expr)' is an existential quantifier. The environment
value for 'ident' must be a list of subenvironments. The
subexpression 'expr' is evaluated in each subenvironment, and if
there exists one subenvironment for which 'expr' is true, the
expression as a whole is true.

Filters are statically type-checked: if a subexpression doesn't have
the appropriate type, Parse will return an error.

*/
package filter
