<!---
The present version of the protocol uses federated consensus mechanism based on a single block generator (elected out of band) and a group of block signers.
-->

# Federated Consensus Protocol

* [Introduction](#introduction)
* [Generator state](#generator-state)
* [Signer state](#signer-state)
* [Algorithms](#algorithms)
  * [Initialize generator](#initialize-generator)
  * [Join new network](#join-new-network)
  * [Accept transaction](#accept-transaction)
  * [Generate block](#generate-block)
  * [Sign block](#sign-block)
  * [Make initial block](#make-initial-block)
  * [Make block](#make-block)

## Introduction

The present version of the protocol uses federated consensus mechanism based on a single *block generator* (elected out of band) and a group of *block signers*.

## Generator state

The block generator maintains, in addition to its [node state](validation.md#node-state):

* A *pending transaction pool*, or simply a *transaction pool*, that is a set of transactions used to construct each block it generates. Transactions can be added to and removed from the transaction pool.
* The *last generated block*, to avoid proposing more than one block at the same height.
* The *generator’s signing key*. All signers recognize the corresponding *verification key* and verify generated blocks using this key before validating and co-signing the block themselves. This key can be replaced by an operator after an out-of-band agreement with other nodes.
* The *maximum issuance window* (in milliseconds), configurable parameter that limits the maximum time field of a transaction containing issuances. This relieves nodes of having to store transactions in the [issuance memory](validation.md#node-state) indefinitely.

Note that the generator can also be a signer, and if so, it also has [signer state](#signer-state).


## Signer state

Each block signer stores, in addition to its node state:

* The *last signed block*. This last signed block can be replaced with a new block in an algorithm below.
* The *generator’s verification key* used to authenticate the block produced by the generator before validating and signing it.
* The *signing key* required by the [consensus program](data.md#consensus-program) in the last signed block. The signing key can be replaced by an operator after an out-of-band agreement with other nodes (by updating the consensus program in one of the future blocks).



## Algorithms

The algorithms below describe the rules for updating a node’s state. Some of the algorithms are used only by other algorithms defined here. Others are entry points — they are triggered by network activity or user input.

Entry Point                                              | When used
---------------------------------------------------------|----------------------------------
[Initialize generator](#initialize-generator)            | A new network is being set up.
[Join new network](#join-new-network)                    | A new network is being set up.
[Accept transaction](#accept-transaction)                | Generator receives a transaction.
[Generate block](#generate-block)                        | Generator produces blocks of transactions at regular intervals.
[Sign block](#sign-block)                                | Block signers co-sign a block produced by a block generator.


### Initialize generator

**Input:** current time.

**Outputs:** none.

**Affects:**

1. current blockchain state,
2. transaction pool.

**Algorithm:**

1. Create a [consensus program](data.md#consensus-program). The contents of this program are a matter of local policy.
2. [Make an initial block](#make-initial-block) with the current time and the created consensus program.
3. Allocate an empty unspent output set.
4. The initial block and these empty sets together constitute the *initial state*.
5. Assign the initial state to the current blockchain state.
6. Create an empty transaction pool.

### Join new network

A new node starts here when joining a new network (with height = 1).

**Input:** block.

**Output:** true or false.

**Affects:** current blockchain state.

**Algorithm:**

1. [Make an initial block](#make-initial-block) with the input block’s timestamp and [consensus program](data.md#consensus-program).
2. The created block must equal the input block; if not, halt and return false.
3. Allocate an empty unspent output set.
4. The initial block and these empty sets together constitute the *initial state*.
5. Assign the initial state to the current blockchain state.
6. Return true.


### Accept transaction

The block generator collects transactions to include in each block it generates. When another node or client has prepared a transaction, it sends the transaction to the generator, which follows this algorithm.

**Inputs:**

1. transaction,
2. current time,
3. current blockchain state.

**Output:** true or false.

**Affects:** transaction pool.

**Algorithm:**

1. [Validate the transaction](validation.md#validate-transaction) with respect to the current blockchain state, but using system timestamp instead of the latest block timestamp; if invalid, halt and return false.
2. If the transaction contains at least one [issuance input with asset version 1](data.md#asset-version-1-issuance-commitment) and a non-empty nonce:
    1. Test that transaction minimum timestamp plus the [maximum issuance window](#generator-state) is greater or equal to the transaction maximum timestamp; if not, halt and return false.
3. Add the transaction to the transaction pool.
4. Return true.

### Generate block

The generator runs this periodically or when the transaction pool reaches a certain size, according to its local policy. It must broadcast the resulting fully-signed block to all other nodes.

**Inputs:**

1. current blockchain state,
2. transaction pool,
3. current time,       
4. last generated block.

**Output:** block.

**Affects:**

1. transaction pool,
2. last generated block.

**Algorithm:**

1. If the last generated block exists with height greater than the current blockchain state, halt and return it.
2. [Make Block](#make-block) with the current blockchain state, the transaction pool, and the current time.
3. For each block signer:
    1. Send the block and the generator’s [signature](data.md#signature) to the signer [asking the signer to sign the block](#sign-block)
    2. Receive a [signature](data.md#signature) from the signer.
    3. Add the signature to the [block witness](data.md#block-witness) program arguments.
4. Replace the last generated block with the new block.
5. [Apply the block](validation.md#apply-block) to the current blockchain state, yielding a new state.
6. Let T be an empty list of transactions.
7. For each transaction in the transaction pool:
    1. [Validate the transaction](validation.md#validate-transaction) with respect to the new state; if invalid, discard it and continue to the next.
    2. Add the transaction to T.
8. Replace the transaction pool with T.
9. Return the block.

Note: steps 6-8 are necessary because the transaction pool is not necessarily fully consumed by the new block.
See also the note in the [Make Block](#make-block) algorithm.

### Sign block


**Inputs:**

1. block,
2. generator’s signature,
3. current blockchain state,
4. last signed block,
5. signing key,
6. system time

**Output:** [signature](data.md#signature) or nothing.

**Affects:** last signed block.

**Algorithm:**

1. Test that the height of the input block is strictly greater than the height of the last signed block; if not, halt and return nothing.
2. Verify the generator’s signature using the generator’s verification key in the current blockchain state. If the signature is invalid, halt and return nothing.
3. [Validate the block](validation.md#validate-block) with respect to the current blockchain state; if invalid, halt and return nothing.
4. Check that the block’s [consensus program](data.md#consensus-program) equals the consensus program in the last signed block; if not, halt and return nothing.
5. Ensure that reserved values and versions are unused. If any of the following conditions are not satisfied, halt and return nothing:
    1. The block version must equal 1.
    2. For every transaction in the block:
        1. Transaction version must equal 1.
        2. [Transaction common witness](data.md#transaction-common-witness) string must be empty.
        3. Every [input witness](data.md#transaction-input-witness) must contain only the fields defined in this version of the protocol (no additional data included).
        4. Every [output witness](data.md#transaction-output-witness) must be empty.
6. Check that the block's timestamp is less than 2 minutes after the system time. If it is not, halt and return nothing.
7. Compute the [block signature hash](data.md#block-signature-hash) for the block.
8. Sign the signature hash with the signing key, yielding a [signature](data.md#signature).
9. Replace the last signed block with the input block.
10. Return the signature.

### Make initial block

**Inputs:**

1. consensus program,
2. time.

**Output**: a block.

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


### Make block

**Inputs:**

1. blockchain state,
2. set of transactions,
3. time.

**Output:** block.

**Algorithm:**

1. Let S be the blockchain state.
2. Let T be an empty list of transactions.
3. For each transaction in the set:
    1. Validate the transaction against S; if it fails, discard the transaction and continue to the next one.
    2. If local policy prohibits the transaction, discard it and continue to the next one.
    3. Add the transaction to T.
    4. Apply the transaction to S, yielding S′.
    5. Replace S with S′.
4. Return a block with the following values:
    1. Version: 1.
    2. Height: 1 + the height of the blockchain state.
    3. Previous block ID: the hash of the blockchain state’s block.
    4. Timestamp: the input time, or the timestamp of the blockchain state increased by 1 millisecond, whichever is greater: `time[n] = max(input_time, time[n-1]+1)`.
    5. [Block commitment](data.md#block-commitment):
        1. [Transactions merkle root](data.md#transactions-merkle-root): [merkle binary tree hash](data.md#merkle-binary-tree) of [witness hashes](data.md#transaction-witness-hash) in T.
        2. [Assets merkle root](data.md#assets-merkle-root): [merkle patricia tree hash](data.md#merkle-patricia-tree) of S.
        3. Consensus program: the input consensus program.
    6. [Block witness](data.md#block-witness): 0x01 0x00 (an empty list of program arguments).
    7. Transaction count: the number of transactions in T.
    8. Transactions: T.

Note: “local policy” in this section gives the generator the ability to exclude
a transaction for any reason. For example, it might apply a fixed size limit
to every block, and stop adding transactions once it reaches that size.
