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
  * [Validate block](#validate-block)
  * [Validate transaction](#validate-transaction)


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

* A [block header](data.md#block-header).
* A *timestamp* equal to the timestamp in the [block header](data.md#block-header).
* The set of [output IDs](data.md#output-id) representing [non-retired](data.md#retired-asset) unspent outputs.
* A *nonce set*: a set of ([Nonce ID](entries.md#nonce), expiration timestamp) pairs. It records recent nonce entries in the state in order to prevent duplicates. Expiration timestamp is used to prune outdated records.


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

1. [Make an initial block](#make-initial-block) with the input block’s timestamp and [consensus program](data.md#consensus-program).
2. The created block must equal the input block; if not, halt and return false.
3. Allocate an empty unspent output set.
4. The initial block and these empty sets together constitute the *initial state*.
5. Assign the initial state to the current blockchain state.
6. Return true.


### Join existing network

A new node starts here when joining a running network (with height > 1). In that case, it does not validate all historical blocks, and the correctness of the blockchain state must be established out of band, for example, by comparing the [block ID](data.md#block-id) to a known-good value.

**Input:** blockchain state.

**Output:** true or false.

**Affects:** current blockchain state.

**Algorithm:**

1. Compute the [assets merkle root](data.md#assets-merkle-root) of the state.
2. The block commitment in the input state must contain the computed assets merkle root; if not, halt and return false.
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
    5. [Block commitment](data.md#block-commitment):
        1. Transactions merkle root: [merkle binary tree hash](data.md#merkle-binary-tree) of the empty list.
        2. Assets merkle root: [merkle patricia tree hash](data.md#merkle-patricia-tree) of the empty list.
        3. Consensus program: the input consensus program.
    6. [Block witness](data.md#block-witness): 0x00 (the empty string).
    7. Transaction count: 0.
    8. Transactions: none.


### Apply block

**Inputs:**

1. block,
2. blockchain state.

**Output:** blockchain state.

**Algorithm:**

1. Evaluate the [consensus program](data.md#consensus-program):
    1. [Create a VM 1](vm1.md#vm-state) with initial state and expansion flag set to `false`.
    2. [Prepare VM](vm1.md#prepare-vm) with program arguments from the block witness.
    4. Set the VM’s program to the consensus program as specified by the blockchain state’s block header.
    5. Execute [Verify Predicate](vm1.md#verify-predicate) operation. If it fails, halt and return false.
2. [Validate the block](entries.md#blockheader-validation) with “previous block header” set to the block header in the current blockchain state; if invalid, halt and return blockchain state unchanged.
3. Let `S` be the input blockchain state.
4. For each transaction in the block, in order:
    1. [Apply the transaction](#apply-transaction) using the input block’s header to blockchain state `S`, yielding a new state `S′`.
    2. If transaction failed to be applied (did not change blockchain state), halt and return the input blockchain state unchanged.
    3. Test that [assets merkle root](data.md#assets-merkle-root) of `S′` is equal to the assets merkle root declared in the block commitment; if not, halt and return blockchain state unchanged.
    4. Replace `S` with `S′`.
5. Remove elements of the nonce set in `S` where the expiration timestamp is less than the block’s timestamp, yielding a new state `S′`.
6. Return the state `S’`.


### Apply transaction

**Inputs:**

1. transaction,
2. block header,
3. blockchain state.

**Output:** blockchain state.

**Algorithm:**

1. [Validate transaction header](entries.md#txheader-validation) with the timestamp and block version of the input block header; if it is not valid, halt and return the input blockchain state unchanged.
2. Let `S` be the input blockchain state.
3. For each visited [nonce entry](entries.md#nonce) in the transaction:
    1. If [nonce ID](entries.md#entry-id) is already stored in the nonce set of the blockchain state, halt and return the input blockchain state unchanged.
    2. Add ([nonce ID](entries.md#entry-id), transaction maxtime) to the nonce set in `S`, yielding a new state `S′`.
    3. Replace `S` with `S′`.
4. For each visited [spend version 1](entries.md#spend-1) in the transaction:
    1. Test that the spent output ID is stored in the set of unspent outputs in `S`. If not, halt and return the input blockchain state unchanged.
    2. Delete the spent output ID from `S`, yielding a new state `S′`.
    3. Replace `S` with `S′`.
5. For each [output version 1](entries.md#output-1) in the transaction header:
    1. Add that output’s [ID](entries.md#entry-id) to `S`, yielding a new state `S′`.
    2. Replace `S` with `S′`.
6. Return `S`.


