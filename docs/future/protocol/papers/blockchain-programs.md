# Blockchain Programs

* [Introduction](#introduction)
* [Chain Virtual Machine](#chain-virtual-machine)
  * [Stack machine basics](#stack-machine-basics)
  * [Run limit](#run-limit)
  * [Instruction set](#instruction-set)
* [Ivy](#ivy)
* [Conclusion](#conclusion)

## Introduction

Chain Protocol enables flexible control over assets by supporting custom logic at three levels:

* **Issuance programs**, that specify the rules for issuing new units of an asset.
* **Control programs**, that specify the rules for spending existing units of an asset.
* **Consensus programs**, that specify the rules for accepting new blocks.

Each program authenticates the data structure in which it is used. Programs run deterministically, use capped memory and time requirements, and can be evaluated in parallel.

Programs are flexible enough to allow implementing:

* a wide range of financial instruments (such as options, bonds, and swaps),
* sophisticated security schemes for holding assets,
* and applications such as offers, order books, and auctions.

This document discusses design and use cases for custom programs on the blockchain.

## Chain Virtual Machine

A program is written in bytecode — instructions for the Chain Virtual Machine (CVM). The CVM is a stack machine: each instruction performs operations on a *data stack*, usually working on the items on top of the stack. All items on the data stack are strings of bytes, although some instructions convert them to and from numbers or booleans in order to perform operations on them. The CVM also has an *alt stack* to simplify stack manipulation.

[sidenote]

Bitcoin, similarly, uses programs as predicates in order to determine whether a given state transition — encoded in a transaction — is authorized. This is different from Ethereum’s approach, in which programs directly compute the resulting state.

[/sidenote]

### Stack machine basics

Let’s take a look at a simple program:

    1 2 ADD 3 EQUAL

This program encodes the predicate `1 + 2 == 3`.

The first two instructions are `PUSHDATA` instructions that push their associated values (encoded within the program) on the data stack.

Next, the `ADD` instruction removes the top two values (`1` and `2`), interprets them as integers, adds them together, and pushes the result (`3`) on the stack.

The next instruction is another `PUSHDATA`. This one pushes the number `3`.

Finally, `EQUAL` removes the top two values (the two copies of the number `3`), compares them byte-by-byte, finds them equal, and so pushes the boolean value `true`.


### Run limit

The CVM’s instruction set is Turing complete. To prevent unbounded use of computational resources, the protocol allows networks to set a *run limit* that a program is not allowed to exceed. Each instruction consumes some of the limit as it runs, according to its *run cost*. Simple instructions have a low cost, while processing-intensive instructions, such as signature checks, are more expensive.

[sidenote]

Both Bitcoin and Ethereum have restrictions that prevent program execution from using excessive time or memory. Chain’s run limit mechanism is similar to Ethereum’s “gas,” except that there is no on-chain accounting for the execution cost of a transaction.

[/sidenote]

The run cost also takes memory usage into account. Adding an item to the stack has a cost based on the size of the item; removing an item from the stack refunds that cost.


### Instruction set

The CVM has some overlaps and similarities with Bitcoin Script, but adds instructions to support additional functionality, including loops, state transitions (through transaction introspection), and program evaluation.

What follows is a summary of the functionality provided by CVM instructions. For a complete list and more precise definitions, see the [VM specification](../specifications/vm1.md).

#### Stack manipulation

Programs may encode bytestrings to push on the data stack using a range of `PUSHDATA` instructions. Instructions such as `DROP`, `DUP`, `SWAP`, `PICK`, and others allow moving stack items around. More complex stack manipulations can be assisted by `TOALTSTACK` and `FROMALTSTACK` instructions that move items between the data stack and an alternate stack.

#### String manipulation

`EQUAL` checks for the equality of two strings. `CAT`, `SUBSTR`, `LEFT`, and `RIGHT` perform operations on strings from the top of the stack. `AND`, `OR`, and `XOR` perform bitwise operations.

#### Arithmetic operations

While all items on the stack are strings, some instructions interpret items as numbers, using 64-bit two’s complement representation.

The CVM deterministically checks for overflows: if the result overflows (e.g. too-large numbers are multiplied), execution immediately fails.

#### Boolean operations

Items on the stack can also be interpreted as booleans. Empty strings and strings consisting of only `0x00` bytes are interpreted as `false`, all others are `true`.

#### Cryptographic operations

The `SHA256` and `SHA3` instructions execute the corresponding hash functions and output 32-byte strings.

The `CHECKSIG` instruction checks the validity of an Ed25519 signature against a given public key and a message hash.

[sidenote]

While similar to Bitcoin instructions, `CHECKSIG` and `CHECKMULTISIG` are generalized to accept an arbitrary message hash. This enables integration with external authoritative data sources and, more importantly, [signature programs](#signature-programs) discussed below.

[/sidenote]

`CHECKMULTISIG` checks an “M-of-N” signing condition using `M` signatures and `N` public keys.

#### Control flow instructions

`VERIFY` pops the top value from the data stack and checks if it is `true`. If it is not, or if there is no top value, the entire program fails.

`JUMPIF` conditionally jumps to another part of the code, based on the current value on top of the stack. This can be used to implement conditionals and loops.

`CHECKPREDICATE` executes a program (written in CVM bytecode) in a separate VM instance. Nested executions are allowed, but the depth is capped by memory cost that is subtracted from the available run limit and refunded when the nested VM instance completes execution.

#### Introspection instructions

The CVM provides operations that, when used in a control or issuance program, introspect parts of a transaction attempting to spend that output.

[sidenote]

The Ethereum VM includes many instructions that provide introspection into the execution environment, including the global mutable state.

In contrast, CVM allows introspection only of the immutable data declared in the transaction, similar to Bitcoin’s `CHECKLOCKTIMEVERIFY` and `CHECKSEQUENCEVERIFY` instructions that check absolute and relative transaction lock times, respectively.

[/sidenote]

`CHECKOUTPUT` allows an input to introspect the outputs of the transaction. This allows it to place restrictions on how the input values are subsequently used. This instruction provides functionality similar to the `CHECKOUTPUTVERIFY` instruction proposed by Malte Möser, Ittay Eyal, and Emin Gün Sirer in their [Bitcoin Covenants](http://fc16.ifca.ai/bitcoin/papers/MES16.pdf) paper. `CHECKOUTPUT` also allows implementing arbitrary state-machines within a UTXO model as was proposed by Oleg Andreev in [Pay-to-Contract](https://github.com/oleganza/bitcoin-papers/blob/master/SmartContractsSoftFork.md) paper.

`MINTIME` and `MAXTIME` allow placing limitations on when an output can be spent. `AMOUNT`, `ASSET`, `PROGRAM`, `REFDATAHASH`, and `INDEX` allow a control program to introspect the input itself.


## Ivy

ChainVM bytecode is much too low-level for users to safely write contracts by hand. For this reason, we developed _Ivy_, a high-level language that compiles to the ChainVM, and which can be used to write programs that control value on Chain blockchain networks.

To learn more about Ivy, and test out Ivy contracts on a Chain blockchain network using the newly released Ivy Playground, click [here](https://chain.com/docs/1.2/ivy-playground/tutorial).

## Conclusion

The Chain Protocol enables flexible control over assets through programmatic conditions that govern both issuance and transfer, as well as integrity of the ledger. Programs are executed by a Chain Virtual Machine with a Turing-complete instruction set. Programs are evaluated as predicates in a restricted, stateless environment that ensures safety and scalability. Programs can use powerful transaction introspection instructions that allow building sophisticated smart contracts and state machines. To make it more efficient to design programs, Chain is developing Ivy, a high-level programming language that compiles to CVM bytecode.
