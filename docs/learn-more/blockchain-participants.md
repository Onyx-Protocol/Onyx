# Participating in a blockchain

## Introduction

This guide will walk you through the three types of blockchain participants:

1. Asset issuer
2. Account manager
3. Blockchain observer

As well as the basic network functions of all blockchain participants:

1. Connecting to a blockchain
2. Receiving blocks
3. Submitting transactions

## Blockchain participants

### Asset issuers

Asset issuers define and issue digital assets into circulation on a blockchain. From governments currencies, to corporate bonds, to loyalty points, to IOUs, to internal deposits, all assets are a guarantee by the issuer of some type of value or rights.

### Account managers

Account managers control asset units on the blockchain. Whether an individual, corporation, financial institution, or government, account managers are cryptographic custodians of digital assets.

### Blockchain observers

Whether an auditor, regulator, or analyst, blockchain observers don't issue or control assets. They simply receive blocks and view blockchain data.

## Basic network functions

### Connecting to a blockchain

When initializing a Chain Core, a participant can connect to an existing blockchain by providing the following information:

1. block generator URL
2. network token
2. blockchain ID

Chain Core will then download all existing blocks from the block generator, in order of creation.

Once all blocks are downloaded, Chain Core will open a persistent connection with the block generator to receive new blocks as they are created.

### Receiving blocks

Each participant in a blockchain independently validates each block it receives from the block generator through the following steps:

1. Validate that the block is signed by the quorum of block signers (as defined in the consensus program of the previous block)
2. Validate each transaction in the block, ensuring each input is properly signed and does not double spend asset units

### Submitting transactions

When submitting a transaction to the client API, Chain Core automatically handles the submission to the block generator. Upon submission, the client API holds the connection open and does not respond with success until it receives a block containing the transaction. Therefore, once a successful response is received from the client API, it is guaranteed that the transaction has been included in a valid block and is now final and immutable on the blockchain.
