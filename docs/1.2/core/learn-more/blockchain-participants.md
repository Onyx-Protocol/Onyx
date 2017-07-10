<!---
This guide will walk you through the three types of blockchain participants, as well as their basic network functions.
-->

# Participating in a blockchain

## Overview

This guide will walk you through the three types of blockchain participants:

1. [Asset issuer](#asset-issuers)
2. [Account manager](#account-managers)
3. [Blockchain observer](#blockchain-observers)

As well as the basic network functions of all blockchain participants:

1. [Connecting to a blockchain](#connecting-to-a-blockchain)
2. [Receiving blocks](#receiving-blocks)
3. [Submitting transactions](#submitting-transactions)

## Blockchain participants

### Asset issuers

Asset issuers define and issue digital assets into circulation on a blockchain. All assets are a guarantee by the issuer of some type of value or rights, from governments currencies, to corporate bonds, to loyalty points, to IOUs, to internal deposits.

### Account managers

Account managers control asset units on the blockchain. Account managers are cryptographic custodians of digital assets and may be individuals, corporations, financial institutions, or governments.

### Blockchain observers

Blockchain observers, such as auditors, regulators, and analysts, don't issue or control assets. They simply receive blocks and view blockchain data.

## Basic network functions

### Connecting to a blockchain

When initializing a Chain Core, a participant can connect to an existing blockchain by providing the following information:

* Block generator URL
* Blockchain ID
* Access token with [cross-core authorization grant](authentication-and-authorization.md#authorization) (not required when connecting to localhost)

Chain Core will begin downloading blockchain data from the block generator. Once the Core is up to date with the network it will receive new blocks as they are created.

### Receiving blocks

Each participant in a blockchain independently validates each block it receives from the block generator through the following steps:

1. Validate that the block is signed by a quorum of block signers (as defined in the consensus program of the previous block)
2. Validate each transaction in the block, ensuring each input is properly signed and does not double-spend asset units

### Submitting transactions

When a transaction is submitted to the Chain Core API, it is automatically relayed to the block generator for inclusion in the next block. The API does not respond until the transaction appears in a block, or an error occurs. Therefore, once a successful response is received from the API, it is guaranteed that the transaction has been included in a valid block and is final and immutable on the blockchain.
