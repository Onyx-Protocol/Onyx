# The Ivy Language

## Introduction

Ivy is Chain’s high-level language for expressing _contracts_, programs that protect value on a blockchain. Each contract _locks_ some value - some number of units of a particular asset - and describes one or more ways to unlock that value.

All value on a Chain blockchain is stored in contracts. When the value in one contract is unlocked, it is only so that it may be locked with one or more new contracts.

A contract protects its value by asking, “Does the transaction trying to spend this value meet my conditions?” Those conditions could be as simple as “the new transaction is signed with Bob’s public key” (for a conventional payment to Bob) but can also be arbitrarily complex. For several examples of Ivy contracts, see [the Ivy Playground Tutorial](tutorial).

This document describes the syntax and semantics of the Ivy language. **This is an early preview of Ivy provided for experimentation only. Not for production use.**

## Contracts and clauses

An Ivy program consists of a contract, defined with the `contract` keyword. A contract definition has the form:

- `contract` _ContractName_ `(` _parameters_ `)` `locks` _value_ `{` _clauses_ `}`

_ContractName_ is an identifier, a name for the contract; _parameters_ is the list of contract parameters, described below; _value_ is an identifier, a name for the value locked by the contract; and _clauses_ is a list of one or more clauses.

Each clause describes one way to unlock the value in the contract, together with any data and/or payments required. A clause looks like this:

- `clause` _ClauseName_ `(` _parameters_ `)` `{` _statements_ `}`

or like this:

- `clause` _ClauseName_ `(` _parameters_ `)` `requires` _payments_ `{` _statements_ `}`

_ClauseName_ is an identifier, a name for the clause; _parameters_ is the list of clause parameters, describe below; _payments_ is a list of required payments, also described below; and _statements_ is a list of one or more statements.

Each statement in a clause is either a `verify`, a `lock`, or an `unlock`. These are further described below.

## Contract and clause parameters

Contract and clause parameters have names and types. A parameter is written as:

- _name_ `:` _TypeName_

and a list of parameters is:

- _name_ `:` _TypeName_ `,` _name_ `:` _TypeName_ `,` ...

Adjacent parameters sharing the same type may be coalesced like so for brevity:

- _name1_ `,` _name2_ `,` ... `:` _TypeName_

so that these two contract declarations are equivalent:

- `contract LockWithMultiSig(key1: PublicKey, key2: PublicKey, key3: PublicKey)`
- `contract LockWithMultiSig(key1, key2, key3: PublicKey)`

Available types are:

- `Amount` `Asset` `Boolean` `Hash` `Integer` `Program` `PublicKey` `Signature` `String` `Time`

These types are described below.

## Required payments, or “clause value”

To unlock the value in a contract, a clause sometimes requires the presence of some other value, such as when dollars are traded for euros. In this case, the clause must use the `requires` syntax to give a name to the required value and to specify its amount and asset type:

- `clause` _ClauseName_ `(` _parameters_ `)` `requires` _name_ `:` _amount_ `of` _asset_

Here, _name_ is an identifier, a name for the value supplied by the transaction unlocking the contract; _amount_ is an expression of type `Amount` (see [expressions](#expressions), below); and _asset_ is an expression of type `Asset`.

Some clauses require two or more payments in order to unlock the contract. Multiple required payments can be specified after `requires` like so:

- ... `requires` _name1_ `:` _amount1_ `of` _asset1_ `,` _name2_ `:` _amount2_ `of` _asset2_ `,` ...

## Statements

The body of a clause contains one or more statements. Contract and clause arguments can be tested in `verify` statements. Contract value can be unlocked in `unlock` statements. Contract and clause value (required payments) can be locked with new programs using `lock` statements.

### Verify statements

A `verify` statement has the form:

- `verify` _expression_

The expression must have `Boolean` type. Every `verify` in a clause must evaluate as true in order for the clause to succeed.

Examples:

- `verify before(deadline)` tests that the transaction unlocking this contract has a timestamp before the given deadline.
- `verify checkTxSig(key, sig)` tests that a given signature matches a given public key _and_ the transaction unlocking this contract.
- `verify newBid > currentBid` tests that one amount is strictly greater than another.

### Unlock statements

Unlock statements only ever have the form:

- `unlock` _value_

where _value_ is the name given to the contract value after the `locks` keyword in the `contract` declaration. This statement releases the contract value for any use; i.e., without specifying the new contract that must lock it.

### Lock statements

Lock statements have the form:

- `lock` _value_ `with` _program_

This locks _value_ (the name of the contract value, or any of the clause’s required payments) with _program_, which is an expression that must have the type `Program`.

## Expressions

Ivy supports a variety of expressions for use in `verify` and `lock` statements as well as the `requires` section of a clause declaration.

- `-` _expr_ negates a numeric expression
- `~` _expr_ inverts the bits in a byte string

Each of the following requires numeric operands (`Integer` or `Amount`) and produces a `Boolean` result:

- _expr1_ `>` _expr2_ tests whether _expr1_ is greater than _expr2_
- _expr1_ `<` _expr2_ tests whether _expr1_ is less than _expr2_
- _expr1_ `>=` _expr2_ tests whether _expr1_ is greater than or equal to _expr2_
- _expr1_ `<=` _expr2_ tests whether _expr1_ is less than or equal to _expr2_
- _expr1_ `==` _expr2_ tests whether _expr1_ is equal to _expr2_
- _expr1_ `!=` _expr2_ tests whether _expr1_ is not equal _expr2_

These operate on byte strings and produce byte string results:

- _expr1_ `^` _expr2_ produces the bitwise XOR of its operands
- _expr1_ `|` _expr2_ produces the bitwise OR of its operands
- _expr1_ `&` _expr2_ produces the bitwise AND of its operands

These operate on numeric operands (`Integer` or `Amount`) and produce a numeric result:

- _expr1_ `+` _expr2_ adds its operands
- _expr1_ `-` _expr2_ subtracts _expr2_ from _expr1_
- _expr1_ `*` _expr2_ multiplies its operands
- _expr1_ `/` _expr2_ divides _expr1_ by _expr2_
- _expr1_ `%` _expr2_ produces _expr1_ modulo _expr2_
- _expr1_ `<<` _expr2_ performs a bitwise left shift on _expr1_ by _expr2_ bits
- _expr1_ `>>` _expr2_ performs a bitwise right shift on _expr1_ by _expr2_ bits

Remaining expression types:

- `(` _expr_ `)` is _expr_
- _expr_ `(` _arguments_ `)` where _arguments_ is a comma-separated list of expressions is a function call; see [functions](#functions) below
- a bare identifier is a variable reference
- `[` _exprs_ `]` where _exprs_ is a comma-separated list of expressions is a list literal (presently used only in `checkTxMultiSig`)
- a sequence of numeric digits optionally preceded by `-` is an integer literal
- a sequence of bytes between single quotes `'...'` is a string literal
- the prefix `0x` followed by 2n hexadecimal digits is also a string literal representing n bytes

### Limitations in Ivy expression syntax

The syntax of Ivy expressions is intentionally limited (compared to conventional programming languages) in an attempt to minimize the chance of writing buggy contracts that could result in misdirected or unrecoverable value.

If you find yourself wanting to write:

```
verify expr1 && expr2
```

write this instead:

```
verify expr1
verify expr2
```

If you’d like to do this:

```
clause execute() {
  verify expr1 || expr2
  ...
```

write this instead:

```
clause execute1() {
  verify expr1
  ...
}
clause execute2() {
  verify expr2
  ...
}
```

## Rules for Ivy contracts

An Ivy contract is correct only if it obeys all of the following rules.

- Identifiers must not collide. A clause parameter, for example, must not have the same name as a contract parameter. (However, two different clauses may reuse the same parameter name; that’s not a collision.)
- Every contract parameter must be used in at least one clause.
- Every clause parameter must be used in its clause.
- Every clause must dispose of the contract value with a `lock` or an `unlock` statement.
- Every clause must also dispose of all clause values, if any, with a `lock` statement for each.
