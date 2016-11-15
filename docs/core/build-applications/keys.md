# Keys

## Introduction

Cryptographic private keys are the primary authorization mechanism on a blockchain. They control both the issuance and transfer of assets. Each transaction is signed using the specific private keys required for the issuance or transfer it proposes, and the signature is checked against the corresponding public keys recorded in the earlier transaction being spent, or the asset type being issued, in order to determine the new transaction’s validity.

In simple cases, an asset or an account will define a single key required for issuance or for transfers. But it’s possible to define multiple keys for different usage patterns or to achieve different levels of security. For example, a high-value asset may be defined with two signing keys, requiring two separate parties to sign each issuance transaction. A joint account may also be defined with two signing keys, requiring only one, from either party, to sign each transfer. The threshold number of signatures required is called a quorum.

In a production environment, private keys are generated within an HSM (hardware security module) and never leave it. Their corresponding public keys are exported for use within Chain Core. In order to issue or transfer asset units on a blockchain, a transaction is created in Chain Core and sent to the HSM for signing. The HSM signs the transaction without ever revealing the private key. Once signed, the transaction can be submitted to the blockchain successfully.

For development environments, Chain Core provides a convenient Mock HSM. The Mock HSM API is identical to the HSM API in [Chain Core Enterprise Edition](https://chain.com/enterprise), providing a seamless transition from development to production. It is important to note that the Mock HSM does not provide the security of a real HSM.

## Overview

This guide will walk you through the basic key operations:

* [Create key](#create-key) (in the Mock HSM)
* [Load key](#load-key) (into the HSM Signer)
* [Sign transaction](#sign-transaction) (with the Mock HSM)

### Sample Code

All code samples in this guide can be viewed in a single, runnable script. Available languages:

- [Java](../examples/java/Keys.java)
- [Ruby](../examples/ruby/keys.rb)

## Create key

Create a new key in the Mock HSM. (Requires a context to have been created with `new Context()`.)

$code create-key ../examples/java/Keys.java ../examples/ruby/keys.rb

## Load key

To be able to sign transactions, load the key into the HSM Signer, which will communicate with the Mock HSM.

$code signer-add-key ../examples/java/Keys.java ../examples/ruby/keys.rb

## Sign transaction

Once a transaction is built, send it to the HsmSigner for signing.

$code sign-transaction ../examples/java/Keys.java ../examples/ruby/keys.rb
