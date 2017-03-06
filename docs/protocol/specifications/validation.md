# Blockchain Validation Protocol

* [Introduction](#introduction)
* [General requirements](#general-requirements)
* [Node state](#node-state)
* [Algorithms](#algorithms)
  * [Join new network](#join-new-network)
  * [Join existing network](#join-existing-network)
  * [Make initial block](#make-initial-block)
  * [Accept block](#accept-block)
  * [Check block is well-formed](#check-block-is-well-formed)
  * [Validate block](#validate-block)
  * [Validate transaction](#validate-transaction)
  * [Validate transaction input](#validate-transaction-input)
  * [Check transaction is well-formed](#check-transaction-is-well-formed)
  * [Apply block](#apply-block)
  * [Apply transaction](#apply-transaction)
  * [Evaluate predicate](#evaluate-predicate)


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
[Join New Network](#join-new-network)                    | A new network is being set up.
[Join Existing Network](#join-existing-network)          | Adding a new node to an already existing network.
[Accept Block](#accept-block)                            | Nodes receive a fully-signed block, validate it and apply it to their state.



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



### Accept block

**Inputs:**

1. block,
2. current blockchain state.

**Output:** true or false.

**Affects:** Current blockchain state.

**Algorithm:**

1. [Evaluate](#evaluate-predicate) the [consensus program](data.md#consensus-program) of the blockchain state as a predicate using VM version 1 with the program arguments of the block witness initializing the data stack.
2. [Validate the block](#validate-block) with respect to the current blockchain state; if invalid, halt and return false.
3. [Apply the block](#apply-block) to the current blockchain state, yielding a new state.
4. Replace the current blockchain state with the new state.
5. Return true.



### Check block is well-formed

**Input:** block.

**Output:** true or false.

**Algorithm:**

1. Test that the block can be parsed as a [block data structure](data.md#block); if not, halt and return false.
2. Test that the block contains the list of transactions (e.g. serialized with [flags](data.md#block-serialization-flags) 0x03); otherwise, halt and return false.
3. For each transaction in the block, test that the [transaction is well-formed](#check-transaction-is-well-formed); if any are not, halt and return false.
4. Compute the [transactions merkle root](data.md#transactions-merkle-root) for the block.
5. Test that the computed merkle tree hash equals the value recorded in the block’s commitment; if not, halt and return false.
6. Return true.


### Validate block

**Inputs:**

1. block,
2. blockchain state.

**Output:** true or false.

**Algorithm:**

1. Test that the [block is well-formed](#check-block-is-well-formed); if not, halt and return false.
2. Test that the block’s version is greater or equal the block version in the blockchain state; if not, halt and return false.
3. Test that the block contains the [block witness](data.md#block-witness) and the list of transactions (e.g. serialized with [flags](data.md#block-serialization-flags) 0x03); otherwise, halt and return false.
4. If the block’s version is 1:
    * Test that the [block commitment](data.md#block-commitment) contains only the fields defined in this version of the protocol; if it contains additional fields, halt and return false.
    * Test that the [block witness](data.md#block-witness) contains only a program arguments field; if it contains additional fields, halt and return false.
5. Test that the block’s [height](data.md#block) is one greater than the height of the blockchain state; if not, halt and return false.
6. Test that the block’s [previous block ID](data.md#block) is equal to the [block ID](data.md#block-id) of the state; if not, halt and return false.
7. Test that the block’s timestamp is greater than the timestamp of the blockchain state; if not, halt and return false.
8. Let S be the input blockchain state.
9. For each transaction in the block, in order:
    1. [Validate the transaction](#validate-transaction) with respect to S; if invalid, halt and return false.
    2. [Apply the transaction](#apply-transaction) to S, yielding a new state S′.
    3. Test that [assets merkle root](data.md#assets-merkle-root) of S′ is equal to the assets merkle root declared in the block commitment; if not, halt and return false.
    4. Replace S with S′.
10. Return true.



### Validate transaction

A transaction is said to be *valid* with respect to a particular blockchain state if it is well formed and if the outputs it attempts to spend exist in the state, and it satisfies the predicates in those outputs. The transaction may or may not be valid with respect to a different blockchain state.

**Inputs:**

1. transaction,
2. blockchain state.

**Output:** true or false.

**Algorithm:**

1. Test that the [transaction is well-formed](#check-transaction-is-well-formed); if not, halt and return false.
2. If the block version in the blockchain state is 1:
    1. Test that transaction version equals 1. If it is not, halt and return false.
3. If the transaction minimum time is greater than zero:
    1. Test that the timestamp of the blockchain state is greater than or equal to the transaction minimum time; if not, halt and return false.
4. If the transaction maximum time is greater than zero:
    1. Test that the timestamp of the blockchain state is less than or equal to the transaction maximum time; if not, halt and return false.
5. If all inputs in transaction are [issuance with asset version 1](data.md#asset-version-1-issuance-commitment), test if at least one of them has a non-empty nonce. If all have empty nonces, halt and return false.
    * Note: this means that transaction uniqueness is guaranteed not only by spending inputs and issuance inputs with non-empty nonce, but also by future inputs of unknown asset versions. The future asset versions will provide rules enforcing transaction uniqueness.
6. For each [issuance input with asset version 1](data.md#asset-version-1-issuance-commitment) and a non-empty nonce, test the following conditions. If any condition is not satisfied, halt and return false:
    1. Both transaction minimum and maximum timestamps are not zero.
    2. State’s timestamp is greater or equal to the transaction minimum timestamp.
    3. State’s timestamp is less or equal to the transaction maximum timestamp.
    4. Input’s [issuance hash](data.md#issuance-hash) does not appear in the state’s nonce set.
7. For every input in the transaction with asset version equal 1, [validate that input](#validate-transaction-input) with respect to the blockchain state; if invalid, halt and return false.
8. Return true.



### Validate transaction input

**Inputs:**

1. transaction input with asset version 1,
2. blockchain state.

**Output:** true or false.

**Algorithm:**

1. If the input is an *issuance*:
    1. Test that the *initial block ID* declared in the witness matches the initial block ID of the current blockchain; if not, halt and return false.
    2. Compute [asset ID](data.md#asset-id) from the initial block ID, asset version 1, and the *VM version* and *issuance program* declared in the witness. If the resulting asset ID is not equal to the declared asset ID in the issuance commitment, halt and return false.
    3. [Evaluate](#evaluate-predicate) its [issuance program](data.md#issuance-program), for the VM version specified in the issuance commitment and with the [input witness](data.md#transaction-input-witness) [program arguments](data.md#program-arguments); if execution fails, halt and return false.
2. If the input is a *spend*:
    1. Check if the state contains the input’s [output ID](data.md#output-id). If the output ID does not exist in the state, halt and return false.
    2. [Evaluate](#evaluate-predicate) the previous output’s control program, for the VM version specified in the previous output and with the [input witness](data.md#transaction-input-witness) program arguments.
    3. If the evaluation returns false, halt and return false.
3. Return true.


### Check transaction is well-formed

**Input:** transaction.

**Output:** true or false.

**Algorithm:**

1. Test that the transaction can be parsed as a [transaction data structure](data.md#transaction); if not, halt and return false.
2. Test that the transaction has at least one input; if not, halt and return false.
3. Ensure that each [input commitment](data.md#transaction-input-commitment) appears only once; if there is a duplicate, halt and return false.
4. If the transaction maximum time is greater than zero test that it is greater than or equal to the minimum time; if not, halt and return false.
5. If transaction version equals 1, check each of the following conditions. If any are not satisfied, halt and return false:
    1. [Transaction common fields](data.md#transaction-common-fields) string must contain only the fields defined in this version of the protocol (no additional data included).
    2. Every [input](data.md#transaction-input) [asset version](data.md#asset-version) must equal 1.
    3. Every [output](data.md#transaction-output) [asset version](data.md#asset-version) must equal 1.
    4. For each input, test that the [input commitment](data.md#transaction-input-commitment) contains only the fields defined in this version of the protocol (no additional data included).
    5. For each output, test that the [output commitment](data.md#transaction-output-commitment) contains only the fields defined in this version of the protocol (no additional data included); if not, halt and return false.
    6. Test that all VM versions in the transaction are 1 (including the VM version in the [issuance input witness](data.md#asset-version-1-issuance-witness)); if not, halt and return false.
    7. Every control program must not contain any [expansion opcodes](vm1.md#expansion-opcodes).
    8. Note: unknown suffixes (additional fields) in [transaction common witness](data.md#transaction-common-witness), [input witnesses](data.md#transaction-input-witness) and [output witnesses](data.md#transaction-output-witness) are not checked here; they are permitted.
6. For inputs and outputs with asset version 1:
    1. For each asset on these inputs and outputs:
        1. Sum the input amounts of that asset and sum the output amounts of that asset.
        2. Test that both input and output sums are less than 2<sup>63</sup>; if not, halt and return false.
        3. Test that the input sum equals the output sum; if not, halt and return false.
        4. Check that there is at least one input with that asset ID; if not, halt and return false.
7. Return true.

Note: requirement for the input and output sums to be below 2<sup>63</sup> implies that all intermediate sums and individual amounts must also be below 2<sup>63</sup> which simplifies implementation that uses native 64-bit unsigned integers.



### Apply block

**Inputs:**

1. block,
2. blockchain state.

**Output:** blockchain state.

**Algorithm:**

1. Let S be the input blockchain state.
2. For each transaction in the block:
    1. [Apply the transaction](#apply-transaction) to S, yielding a new state S′.
    2. Replace S with S′.
3. Replace the block header in S with the input block header, yielding a new state S′.
4. Replace S with S′.
5. Remove elements of the nonce set in S where the expiration timestamp is less than the block’s timestamp, yielding a new state S′.
6. Return S′.

### Apply Transaction

**Inputs:**

1. transaction,
2. blockchain state.

**Output:** blockchain state.

**Algorithm:**

1. For each spend input with asset version 1 in the transaction:
    1. Delete the previous [output ID](data.md#output-id) from S, yielding a new state S′.
    2. Replace S with S′.
2. For each output with asset version 1 in the transaction:
    1. Add that output’s [output ID](data.md#output-id) to S, yielding a new state S′.
    2. Replace S with S′.
3. For all [asset version 1 issuance inputs](data.md#asset-version-1-issuance-commitment) with non-empty *nonce* string:
    1. Compute the [issuance hash](data.md#issuance-hash) H.
    2. Add (H, transaction maximum timestamp) to the nonce set in S, yielding a new state S′.
    3. Replace S with S′.
4. Return S.


### Evaluate predicate

**Inputs:**

1. VM version,
2. program,
3. list of program arguments.

**Output:** true or false.

**Algorithm:**

1. If the [VM version](vm1.md#versioning) is > 1, halt and return true.
2. [Create a VM with initial state](vm1.md#vm-state).
3. [Prepare VM](vm1.md#prepare-vm).
4. Set the VM’s program to the predicate program and execute [Verify Predicate](vm1.md#verify-predicate) operation. If it fails, halt and return false.
5. Return true.


