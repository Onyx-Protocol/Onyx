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

1. Each block signer generates a public/private keypair for block signing, as well as access credentials that the generator can use to send messages to the block signer's core.
2. Out of band, each block signer sends its public key, access credentials, and core URL to the block generator.
3. In most cases, the block generator creates its own public/private keypair for block signing.
4. The block generator configures its core with each block signer's core URL, access credentials, and public keys.
5. The block generator creates an initial consensus program (from the public keys and quorum of the block signers), and creates the first block, including the consensus program.
6. The block generator generates access credentials for each block signer. Block signers can use these credentials to send requests to the block generator's core.
7. Out of band, the block generator distributes access credentials and its core URL to all block signers. It also sends the hash of the initial block, known as the blockchain ID.
8. Using the generator's access credentials, core URL, and blockchain ID, the block signers can finish configuring their cores.

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

Instances of Chain Core must be configured with the correct access credentials (either access tokens or X.509 client certificates) in order to communicate with each other. In particular:

- All participants must make requests to the block generator using a credential that has access to the generator's `crosscore` policy.
- The generator must make requests to block signers using credentials that have access to each signer's `crosscore-signblock` policy, respectively.

The [Authentication and Authorization guide](authentication-and-authorization.md) contains more detail on how to create and configure credentials and policies.

### Adding/removing blockchain operators

To adjust the set of blockchain operators, a change must be made to the consensus program, and a quorum of existing block signers must agree to the change. This procedure requires tools still under development and coming soon.
