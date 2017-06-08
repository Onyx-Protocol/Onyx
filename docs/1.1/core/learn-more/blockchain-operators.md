<!---
This guide will walk you through the basic functions of the blockchain operators: creating a new blockchain, creating blocks, permissioning the network, and adding/removing blockchain operators.
-->

# Operating a blockchain

## Introduction

The operators of a blockchain perform four basic functions:

1. Determine who can participate in the blockchain
2. Gather valid transactions from participants
2. Generate and sign blocks of valid transactions
3. Distribute blocks to participants

One of the blockchain operators is designated as the block generator. The others are designated as block signers. Together, they are responsible for creating new blocks.

Each block contains a consensus program that defines the requirements for creating the next valid block. The consensus program specifies the public key of the block generator (whose signature is required on the next block) and the public keys of a set of block signers with a quorum of signatures that are required on the next block.

## Overview

This guide will walk you through the basic functions of the blockchain operators:

1. [Creating a new blockchain](#creating-a-new-blockchain)
2. [Creating blocks](#creating-blocks)
3. [Permissioning the network](#network-permissions)
4. [Adding/removing blockchain operators](#adding-removing-blockchain-operators)

### Creating a new blockchain

To create a new blockchain, the blockchain operators must coordinate to create the initial consensus program and generate the first block (at height 0). The process is as follows:

1. Each block signer initializes a Chain Core as a block signer, creating a network token and a private/public keypair.
2. Each block signer distributes their block signer URL, network token, and public key to the block generator out of band.
3. The block generator initializes a Chain Core as a block generator, creating a private/public keypair.
4. The block generator configures the URL, network token, and public key for each block signer in Chain Core settings.
5. The block generator creates the initial consensus program (from its public key and the public keys and quorum of the block signers) in Chain Core settings.
6. The block generator creates the first block, including the initial consensus program, which is automatically distributed to each block signer.

Note: The Chain Core dashboard does not yet support block signer configuration. However, you can use the Chain Core command line tools to configure block generator and block signers manually. See the [block signing guide](configure-block-signers.md).

### Creating blocks

#### Block generator

The block generator is responsible creating blocks at a defined interval through the following steps:

1. Accept transactions from participants
2. Validate each transaction to ensure it is properly signed and does not double spend asset units
3. Generate a block of valid transactions
4. Sign the block
5. Gather signatures from the required quorum of block signers
6. Distribute the block to participants

#### Block signers

Once the block generator has generated a proposed block, each block signer (up to the quorum) will sign the block through the following steps:

1. Accept a proposed block from the block generator
2. Validate the block, ensuring that it has never signed a block at the same height
2. Validate each transaction in the block, ensuring each input is properly signed and does not double spend asset units
4. Sign the block
5. Return the signed block to the block generator

### Network permissions

A blockchain can be configured to require network tokens in order to connect to the block generator to submit transactions and receive blocks. The block generator can create a unique network token for each participant that can be revoked at any time.

### Adding/removing blockchain operators

To adjust the set of blockchain operators, a change must be made to the consensus program, and a quorum of existing block signers must agree to the change. This procedure requires tools still under development and coming soon.
