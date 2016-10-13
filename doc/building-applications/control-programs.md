# Control Programs

## Introduction

A control program is the mechanism that secures asset units on a blockchain. When you first issue units of an asset, you issue them into a control program. When you spend units of an asset, you spend them from an existing control program to a new control program. When you retire units of an asset, you spend them from an existing control program to a special retirement control program that can never be spent.

Each output in a transaction contains a single control program. Each control program consists of set of predicates that must be satisfied in order to spend the output (i.e. use the output as an input to a new transaction).

## Account control programs

The most basic type of control program is an account control program, which defines a set of keys and a quorum of signatures required to spend asset units. When you create an account, you provide as set of root keys and a quorum. Then each time you deposit assets into an account, Chain Core derives a new set of child public keys from the account root keys and creates a unique, one-time-use account control program requiring the quorum of signatures you specified.

Although all control programs in one account are controlled by keys derived from the same root keys, it is impossible for other participants on the blockchain to recognize any relationship between them. This technique (known as heirarchical deterministic key derivation) ensures that only the participant on the blockchain with whom you transact will know that a specific control program is yours. To everyone else, the creator of the control program will be unknown. For more information about key derivation, see [HD keys](#).

### Example

If Alice wishes to be paid gold by an external party (Bob), she first creates a new control program in her account:

$code /doc/examples/java/ControlPrograms.java create-control-program

She can then deliver the control program to Bob (out of band), who will provide it to the Transaction.QueryBuilder:

$code /doc/examples/java/ControlPrograms.java build-transaction

## Retirement control programs

A retirement control program is a very simple control program with a single predicate: `FAIL`. This ensures that asset units sent to this type of control can never be spent, and thus, are removed from circulation on the blockchain.

### Example

To retire units of gold from Alice's account, we use the `SpendFromAccount` and `Retire` actions on the `Transaction.QueryBuilder`, which prompts Chain Core to create the retirement control program and spent to it from Alice's account.

$code /doc/examples/java/ControlPrograms.java retire

## Custom control programs

The Chain Core virtual machine supports custom control programs built from the supported set of [program predicates](#). We are [currently developing](/doc/getting-started/product-roadmap) a high level language that will enable developers to write custom control programs in Chain Core. Additionally, we work directly with our enterprise customers to design, audit, and implement custom control programs for production deployment. For more information, visit [Chain Core for Enterprise](#) or [contact us](#).

[Download Code](/doc/examples/java/ControlPrograms.java)
