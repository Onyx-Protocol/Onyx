package compiler

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	chainjson "chain/encoding/json"
)

// Contract is a compiled Ivy contract.
type Contract struct {
	// Name is the contract name.
	Name string `json:"name"`

	// Params is the list of contract parameters.
	Params []*Param `json:"params,omitempty"`

	// Clauses is the list of contract clauses.
	Clauses []*Clause `json:"clauses"`

	// Value is the name of the value locked by the contract.
	Value string `json:"value"`

	// Body is the optimized bytecode of the contract body. This is not
	// a complete program!  Use instantiate to turn this (plus some
	// arguments) into a program.
	Body chainjson.HexBytes `json:"body_bytecode"`

	// Opcodes is the human-readable string of opcodes corresponding to
	// Body.
	Opcodes string `json:"body_opcodes,omitempty"`

	// Recursive tells whether this contract calls itself.  (This is
	// used to select between two possible instantiation options.)
	Recursive bool `json:"recursive"`

	// Pre-optimized list of instruction steps, with stack snapshots.
	Steps []Step `json:"-"`

	Tokens []token `json:"-"`
}

// Param is a contract or clause parameter.
type Param struct {
	// Name is the parameter name.
	Name string `json:"name"`

	// Type is the declared parameter type.
	Type typeDesc `json:"declared_type"`

	// InferredType, if available, is a more-specific type than Type,
	// inferred from the logic of the contract.
	InferredType typeDesc `json:"inferred_type,omitempty"`

	Tokens []token `json:"-"`
}

// Clause is a compiled contract clause.
type Clause struct {
	// Name is the clause name.
	Name string `json:"name"`

	// Params is the list of clause parameters.
	Params []*Param `json:"params,omitempty"`

	// Reqs is the list of requirements (from the clause's "requires"
	// section).
	Reqs []*ClauseReq `json:"reqs,omitempty"`

	Statements []Statement

	// MinTimes is the list of expressions passed to after() in this
	// clause.
	MinTimes []string `json:"mintimes,omitempty"`

	// MaxTimes is the list of expressions passed to before() in this
	// clause.
	MaxTimes []string `json:"maxtimes,omitempty"`

	// HashCalls is the list of hash functions and their arguments used
	// in this clause.
	HashCalls []HashCall `json:"hash_calls,omitempty"`

	// Values is the list of values unlocked or relocked in this clause.
	Values []ValueInfo `json:"values"`

	// Contracts is the list of contracts called by this clause.
	Contracts []string `json:"contracts,omitempty"`

	Tokens []token `json:"-"`
}

// HashCall describes a call to a hash function.
type HashCall struct {
	// HashType is "sha3" or "sha256".
	HashType string `json:"hash_type"`

	// Arg is the expression passed to the hash function.
	Arg string `json:"arg"`

	// ArgType is the type of Arg.
	ArgType string `json:"arg_type"`
}

// ClauseReq describes a payment requirement of a clause (one of the
// things after the "requires" keyword).
type ClauseReq struct {
	Name string `json:"name"`

	assetExpr, amountExpr Expression

	// Asset is the expression describing the required asset.
	Asset string `json:"asset"`

	// Amount is the expression describing the required amount.
	Amount string `json:"amount"`

	Tokens []token `json:"-"`
}

type Statement interface {
	countVarRefs(map[string]int)
}

type VerifyStatement struct {
	Expr Expression

	Tokens []token `json:"-"`
}

func (s VerifyStatement) countVarRefs(counts map[string]int) {
	s.Expr.countVarRefs(counts)
}

type LockStatement struct {
	Locked  Expression
	Program Expression

	// Added as a decoration, used by CHECKOUTPUT
	index int64

	Tokens []token `json:"-"`
}

func (s LockStatement) countVarRefs(counts map[string]int) {
	s.Locked.countVarRefs(counts)
	s.Program.countVarRefs(counts)
}

type UnlockStatement struct {
	Expr Expression

	Tokens []token `json:"-"`
}

func (s UnlockStatement) countVarRefs(counts map[string]int) {
	s.Expr.countVarRefs(counts)
}

type Expression interface {
	String() string
	typ(*environ) typeDesc
	countVarRefs(map[string]int)
}

type BinaryExpr struct {
	left, right Expression
	op          *binaryOp

	Tokens []token `json:"-"`
}

func (e BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", e.left, e.op.op, e.right)
}

func (e BinaryExpr) typ(*environ) typeDesc {
	return e.op.result
}

func (e BinaryExpr) countVarRefs(counts map[string]int) {
	e.left.countVarRefs(counts)
	e.right.countVarRefs(counts)
}

type UnaryExpr struct {
	op   *unaryOp
	expr Expression

	Tokens []token `json:"-"`
}

func (e UnaryExpr) String() string {
	return fmt.Sprintf("%s%s", e.op.op, e.expr)
}

func (e UnaryExpr) typ(*environ) typeDesc {
	return e.op.result
}

func (e UnaryExpr) countVarRefs(counts map[string]int) {
	e.expr.countVarRefs(counts)
}

type CallExpr struct {
	fn   Expression
	args []Expression

	Tokens []token `json:"-"`
}

func (e CallExpr) String() string {
	var argStrs []string
	for _, a := range e.args {
		argStrs = append(argStrs, a.String())
	}
	return fmt.Sprintf("%s(%s)", e.fn, strings.Join(argStrs, ", "))
}

func (e CallExpr) typ(env *environ) typeDesc {
	if b := referencedBuiltin(e.fn); b != nil {
		switch b.name {
		case "sha3":
			if len(e.args) == 1 {
				switch e.args[0].typ(env) {
				case strType:
					return sha3StrType
				case pubkeyType:
					return sha3PubkeyType
				}
			}

		case "sha256":
			if len(e.args) == 1 {
				switch e.args[0].typ(env) {
				case strType:
					return sha256StrType
				case pubkeyType:
					return sha256PubkeyType
				}
			}
		}

		return b.result
	}
	if e.fn.typ(env) == predType {
		return boolType
	}
	if e.fn.typ(env) == contractType {
		return progType
	}
	return nilType
}

func (e CallExpr) countVarRefs(counts map[string]int) {
	e.fn.countVarRefs(counts)
	for _, a := range e.args {
		a.countVarRefs(counts)
	}
}

type VarRef string

func (v VarRef) String() string {
	return string(v)
}

func (e VarRef) typ(env *environ) typeDesc {
	if entry := env.lookup(string(e)); entry != nil {
		return entry.t
	}
	return nilType
}

func (e VarRef) countVarRefs(counts map[string]int) {
	counts[string(e)]++
}

type BytesLiteral []byte

func (e BytesLiteral) String() string {
	return "0x" + hex.EncodeToString([]byte(e))
}

func (BytesLiteral) typ(*environ) typeDesc {
	return "String"
}

func (BytesLiteral) countVarRefs(map[string]int) {}

type IntegerLiteral int64

func (e IntegerLiteral) String() string {
	return strconv.FormatInt(int64(e), 10)
}

func (IntegerLiteral) typ(*environ) typeDesc {
	return "Integer"
}

func (IntegerLiteral) countVarRefs(map[string]int) {}

type BooleanLiteral bool

func (e BooleanLiteral) String() string {
	if e {
		return "true"
	}
	return "false"
}

func (BooleanLiteral) typ(*environ) typeDesc {
	return "Boolean"
}

func (BooleanLiteral) countVarRefs(map[string]int) {}

type ListExpr []Expression

func (e ListExpr) String() string {
	var elts []string
	for _, elt := range e {
		elts = append(elts, elt.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(elts, ", "))
}

func (ListExpr) typ(*environ) typeDesc {
	return "List"
}

func (e ListExpr) countVarRefs(counts map[string]int) {
	for _, elt := range e {
		elt.countVarRefs(counts)
	}
}
