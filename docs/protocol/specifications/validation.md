# Blockchain Validation Protocol

* [Introduction](#introduction)
* [General requirements](#general-requirements)
* [Node state](#node-state)
* [Algorithms](#algorithms)
  * [Join new network](#join-new-network)
  * [Join existing network](#join-existing-network)
  * [Make initial block](#make-initial-block)
  * [Apply block](#apply-block)
  * [Apply transaction](#apply-transaction)


## Introduction

This document describes all the algorithms involved in participating as a node in a blockchain network, including their interfaces and requirements for persistent state.

## General requirements

### Interfaces

Each algorithm specifies its interface to the outside world under the labels *inputs*, *outputs*, *references*, and *affects*. *Input* is data that must be explicitly provided to the algorithm by its invoking context (either another algorithm or a user). *Output* is data that is explicitly returned from the algorithm to its invoking context. *References* lists elements of the persistent state (described above) that might be read. *Affects* lists elements of the persistent state that can be modified.

### Optimization

A conforming implementation must behave as if it is following the algorithms described below. However, for the sake of efficiency or convenience, it is permitted to take other actions instead as long as they yield the same result. For example, a conforming implementation might "memoize" the result of a computation rather than recomputing it multiple times, or it might perform the steps of an algorithm in a different but equivalent order.

### Serializability

A conforming implementation must be serializable with respect to these algorithms. That is, it can execute them concurrently in parallel, but it must produce the same output and side effects as if they had run serially in some order.

This requirement also implies that all side effects together must be atomic for each algorithm.

### Node state

All nodes store a *current blockchain state*, which can be replaced with a new blockchain state.

A *blockchain state* comprises:

* A [block header](blockchain.md#block-header).
* A *timestamp* equal to the timestamp in the [block header](blockchain.md#block-header).
* A *UTXO set*: a set of output [IDs](blockchain.md#entry-id) representing unspent [outputs](blockchain.md#output-1).
* A *nonce set*: a set of ([Nonce ID](blockchain.md#nonce), expiration timestamp) pairs. It records recent nonce entries in the state in order to prevent duplicates. Expiration timestamp is used to prune outdated records.


## Algorithms

The algorithms below describe the rules for updating a node’s state. Some of the algorithms are used only by other algorithms defined here. Others are entry points — they are triggered by network activity or user input.

Entry Point                                              | When used
---------------------------------------------------------|----------------------------------
[Join new network](#join-new-network)                    | A new network is being set up.
[Join existing network](#join-existing-network)          | Adding a new node to an already existing network.
[Apply block](#apply-block)                              | Nodes receive a fully-signed block, validate it and apply it to their state.



### Join new network

A new node starts here when joining a new network (with height = 1).

**Inputs:**

1. consensus program,
2. time.

**Output:** true or false.

**Affects:** current blockchain state.

**Algorithm:**

1. [Make an initial block](#make-initial-block) with the input block’s timestamp and [consensus program](blockchain.md#block-header).
2. The created block must equal the input block; if not, halt and return false.
3. Allocate an empty unspent output set.
4. The initial block and these empty sets together constitute the *initial state*.
5. Assign the initial state to the current blockchain state.
6. Return true.


### Join existing network

A new node starts here when joining a running network (with height > 1). In that case, it does not validate all historical blocks, and the correctness of the blockchain state must be established out of band, for example, by comparing the [block ID](blockchain.md#block-id) to a known-good value.

**Input:** blockchain state.

**Output:** true or false.

**Affects:** current blockchain state.

**Algorithm:**

1. Compute the [assets merkle root](blockchain.md#assets-merkle-root) of the state.
2. The block header in the input state must contain the computed assets merkle root; if not, halt and return false.
3. Assign the input state to the current blockchain state.
4. Return true.


### Make initial block

**Inputs:**

1. consensus program,
2. time.

**Output**: block.

**Algorithm:**

1. Return a block with the following values:
    1. Version: 1.
    2. Height: 1.
    3. Previous block ID: 32 zero bytes.
    4. Timestamp: the input time.
    5. Transactions merkle root: [merkle binary tree hash](blockchain.md#merkle-binary-tree) of the empty list.
    6. Assets merkle root: [merkle patricia tree hash](blockchain.md#merkle-patricia-tree) of the empty list.
    7. Consensus program: the input consensus program.
    8. Arguments: an empty list.
    9. Transaction count: 0.
    10. Transactions: none.


### Apply block

**Inputs:**

1. block,
2. blockchain state.

**Output:** blockchain state.

**Algorithm:**

1. [Validate the block header](blockchain.md#block-header-validation) with “previous block header” set to the block header in the current blockchain state; if invalid, halt and return blockchain state unchanged.
2. Let `S` be the input blockchain state.
3. For each transaction in the block, in order:
    1. [Apply the transaction](#apply-transaction) using the input block’s header to blockchain state `S`, yielding a new state `S′`.
    2. If transaction failed to be applied (did not change blockchain state), halt and return the input blockchain state unchanged.
    3. Replace `S` with `S′`.
4. Test that [assets merkle root](blockchain.md#assets-merkle-root) of `S` is equal to the assets merkle root declared in the block header; if not, halt and return blockchain state unchanged.
5. Remove elements of the nonce set in `S` where the expiration timestamp is less than the block’s timestamp, yielding a new state `S′`.
6. Return the state `S’`.


### Apply transaction

**Inputs:**

1. transaction,
2. block header,
3. blockchain state.

**Output:** blockchain state.

**Algorithm:**

1. Validate transaction using the checks below. If any check fails, halt and return the input blockchain state unchanged.
    1. If the block header version is 1, verify that transaction version is equal to 1.
    2. If the `Mintime` is greater than zero: verify that it is less than or equal to the block header timestamp.
    3. If the `Maxtime` is greater than zero: verify that it is greater than or equal to the block header timestamp.
    4. [Validate transaction header](blockchain.md#transaction-header-validation).
2. Let `S` be the input blockchain state.
3. For each visited [nonce entry](blockchain.md#nonce) in the transaction:
    1. If [nonce ID](blockchain.md#entry-id) is already stored in the nonce set of the blockchain state, halt and return the input blockchain state unchanged.
    2. Add ([nonce ID](blockchain.md#entry-id), nonce maxtime) to the nonce set in `S`, yielding a new state `S′`.
    3. Replace `S` with `S′`.
4. For each visited [spend version 1](blockchain.md#spend-1) in the transaction:
    1. Test that the spent output ID is stored in the set of unspent outputs in `S`. If not, halt and return the input blockchain state unchanged.
    2. Delete the spent output ID from `S`, yielding a new state `S′`.
    3. Replace `S` with `S′`.
5. For each [output version 1](blockchain.md#output-1) in the transaction header:
    1. Add that output’s [ID](blockchain.md#entry-id) to `S`, yielding a new state `S′`.
    2. Replace `S` with `S′`.
6. Return `S`.


