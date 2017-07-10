<!---
Ivy is Chain’s high-level language for expressing contracts, programs that protect value on a blockchain.
-->

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
- _expr_ `(` _arguments_ `)` is a function call, where _arguments_ is a comma-separated list of expressions; see [functions](#functions) below
- a bare identifier is a variable reference
- `[` _exprs_ `]` is a list literal, where _exprs_ is a comma-separated list of expressions (presently used only in `checkTxMultiSig`)
- a sequence of numeric digits optionally preceded by `-` is an integer literal
- a sequence of bytes between single quotes `'...'` is a string literal
- the prefix `0x` followed by 2n hexadecimal digits is also a string literal representing n bytes

## Functions

Ivy includes several built-in functions for use in `verify` statements and elsewhere.

- `abs(n)` takes a number and produces its absolute value.
- `min(x, y)` takes two numbers and produces the smaller one.
- `max(x, y)` takes two numbers and produces the larger one.
- `size(s)` takes an expression of any type and produces its `Integer` size in bytes.
- `concat(s1, s2)` takes two strings and concatenates them to produce a new string.
- `concatpush(s1, s2)` takes two strings and produces the concatenation of `s1` followed by the Chain VM opcodes needed to push `s2` onto the Chain VM stack. This is typically used to construct new VM programs out of pieces of other ones. See [the Chain VM specification](/protocol/specifications/vm1#catpushdata).
- `before(t)` takes a `Time` and returns a `Boolean` telling whether the unlocking transaction has a timestamp prior to `t`.
- `after(t)` takes a `Time` and returns a `Boolean` telling whether the unlocking transaction has a timestamp later than `t`.
- `sha3(s)` takes a byte string and produces its SHA3-256 hash (with Ivy type `Hash`).
- `sha256(s)` takes a byte string and produces its SHA-256 hash (with Ivy type `Hash`).
- `checkTxSig(key, sig)` takes a `PublicKey` and a `Signature` and returns a `Boolean` telling whether `sig` matches both `key` _and_ the unlocking transaction.
- `checkTxMultiSig([key1, key2, ...], [sig1, sig2, ...])` takes one list-literal of `PublicKeys` and another of `Signatures` and returns a `Boolean` that is true only when every `sig` matches both a `key` _and_ the unlocking transaction. Ordering matters: not every key needs a matching signature, but every signature needs a matching key, and those must be in the same order in their respective lists.

## Rules for Ivy contracts

An Ivy contract is correct only if it obeys all of the following rules.

- Identifiers must not collide. A clause parameter, for example, must not have the same name as a contract parameter. (However, two different clauses may reuse the same parameter name; that’s not a collision.)
- Every contract parameter must be used in at least one clause.
- Every clause parameter must be used in its clause.
- Every clause must dispose of the contract value with a `lock` or an `unlock` statement.
- Every clause must also dispose of all clause values, if any, with a `lock` statement for each.

## Examples

Here is `LockWithPublicKey`, one of the simplest possible contracts. Armed with the information in this document we can understand in detail how it works.

```
contract LockWithPublicKey(publicKey: PublicKey) locks value {
  clause spend(sig: Signature) {
    verify checkTxSig(publicKey, sig)
    unlock value
  }
}
```

The name of this contract is `LockWithPublicKey`. It locks some value, called `value`. The transaction locking the value must specify an argument for `LockWithPublicKey`’s one parameter, `publicKey`.

`LockWithPublicKey` has one clause, which means one way to unlock `value`: `spend`, which requires a `Signature` as an argument.

The `verify` in `spend` checks that `sig`, the supplied `Signature`, matches both `publicKey` and the new transaction trying to unlock `value`. If that succeeds, `value` is unlocked.

Here is a more challenging example: the `LoanCollateral` contract from [the Ivy playground](tutorial).

```
contract LoanCollateral(assetLoaned: Asset,
                        amountLoaned: Amount,
                        repaymentDue: Time,
                        lender: Program,
                        borrower: Program) locks collateral {
  clause repay() requires payment: amountLoaned of assetLoaned {
    lock payment with lender
    lock collateral with borrower
  }
  clause default() {
    verify after(repaymentDue)
    lock collateral with lender
  }
}
```

The name of this contract is `LoanCollateral`. It locks some value called `collateral`. The transaction locking `collateral` with this contract must specify arguments for `LoanCollateral`’s five parameters: `assetLoaned`, `amountLoaned`, `repaymentDue`, `lender`, and `borrower`.

The contract has two clauses, or two ways to unlock `collateral`:

- `repay` requires no data but does require payment of `amountLoaned` units of `assetLoaned`
- `default` requires no payment or data

The intent of this contract is that `lender` has loaned `amountLoaned` units of `assetLoaned` to `borrower`, secured by `collateral`; and if the loan is repaid to `lender`, the collateral is returned to `borrower`, but if the repayment deadline passes, `lender` is entitled to claim `collateral` for him or herself.

The statements in `repay` send the payment to the lender and the collateral to the borrower with a simple pair of `lock` statements. Recall that “sending” value “to” a blockchain participant actually means locking the payment with a program that allows the recipient to unlock it.

The `verify` in `default` ensures that the deadline has passed and, if it has, the `lock` statement locks `collateral` with `lender`. Note that this does not happen automatically when the deadline passes. The lender (or someone) must explicitly unlock `collateral` by constructing a new transaction that invokes the `default` clause of `LoanCollateral`.
