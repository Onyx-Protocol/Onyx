# Ivy Playground Docs

## Introduction

Ivy is Chain’s high-level language for expressing _contracts_, programs that protect value on a blockchain. Each contract _locks_ some value - some number of units of a particular asset - and describes one or more ways to unlock that value.

All value on a Chain blockchain is stored in contracts. Unlocking the value in one contract only ever happens while simultaneously locking the value with one or more new contracts.

A contract protects its value by asking, “Does the transaction trying to spend this value meet my conditions?” Those conditions could be as simple as “the new transaction is signed with Bob’s public key” (for a conventional payment to Bob) but can also be arbitrarily complex. For several examples of Ivy contracts, see [the Ivy Playground Tutorial](tutorial).

This document describes the syntax and semantics of the Ivy language. **This is an early preview of Ivy provided for experimentation only. Not for production use.**

## Contracts and clauses

An Ivy program consists of a contract, defined with the `contract` keyword. A contract definition has the form:

`contract` _ContractName_ `(` _parameters_ `)` `locks` _value_ `{` _clauses_ `}`

_ContractName_ is an identifier, a name for the contract; _parameters_ is the list of contract parameters, described below; _value_ is an identifier, a name for the value locked by the contract; and _clauses_ is a list of one or more clauses.

Each clause describes one way to unlock the value in the contract, together with any data and/or payments required. A clause looks like this:

`clause` _ClauseName_ `(` _parameters_ `)` `{` _statements_ `}`

or like this:

`clause` _ClauseName_ `(` _parameters_ `)` `requires` _payments_ `{` _statements_ `}`

_ClauseName_ is an identifier, a name for the clause; _parameters_ is the list of clause parameters, describe below; _payments_ is a list of required payment, also described below; and _statements_ is a list of one or more statements.

Each statement in a clause is either a `verify`, a `lock`, or an `unlock`. These are further described below.

## Contract and clause parameters

Contract and clause parameters have names and types. A parameter is written as:

_name_ `:` _TypeName_

and a list of parameters is:

_name_ `:` _TypeName_ `,` _name_ `:` _TypeName_ `,` ...

Adjacent parameters sharing the same type may be coalesced like so for brevity:

_name1_ `,` _name2_ `,` ... `:` _TypeName_

so that these two contract declarations are equivalent:

`contract LockWithMultiSig(key1: PublicKey, key2: PublicKey, key3: PublicKey)`

`contract LockWithMultiSig(key1, key2, key3: PublicKey)`

Available types are:

```
Amount Asset Boolean Hash Integer Program PublicKey Signature String Time
```

These types are described below.
