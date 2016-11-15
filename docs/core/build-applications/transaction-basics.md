# Transaction Basics

## Introduction

A blockchain consists of an immutable set of cryptographically linked transactions. Each transaction contains one or more **inputs**, and one or more **outputs**.

An input either issues new units of an asset, or transfers existing units by naming an output of an earlier transaction as the source for the transfer.

Outputs take the asset units from the inputs and define how they will be allocated. A single output indicates an asset amount along with a control program that specifies how that amount can be spent in the future. Or, an output can retire units of the asset, removing those units from circulation.

A transaction can have many inputs and outputs that consist of many different types of asset, many different sources, and many different destinations. All actions within a transaction (issuing, spending, controlling, and retiring assets) occur simultaneously, as a single, atomic operation. There is never any point where a transaction is only partially applied.

### Asset balancing

Within a transaction, the total amount of assets in the inputs must equal the total amount of assets in the outputs. To create new asset units, we issue in an input and control them with one or more outputs. To transfer asset units, we spend them in an input and control them with one or more outputs. To retire assets units, we spend them in an input and retire them in an output.

Any combination of inputs and outputs can be used in a single transaction, as long as what goes in is what comes out.

### Consuming entire unspent outputs

When creating an input that spends an earlier transaction’s output, the entire amount in that output must be consumed. If you don't want to transfer the entire amount of the output to your intended recipient, you must make change back to your account. This is analogous to cash--if you have a twenty-dollar bill and want to spent ten dollars, you need to get change.

As a result, spending a single input often requires two outputs--one output for the intended recipient, and one output for change back to the account where the asset units came from. In general, Chain Core will automatically manage change outputs for you, so you don't have to worry about this in your application code.

### Combining unspent outputs

Some payments may require more asset units than are available in any single unspent output you control. When spending from an account, the Chain Core will automatically select unspent outputs to satisfy your payment as long as the account controls enough units of the asset in total.

## Overview

This guide describes the structure of transactions, and how to use the Chain Core API and SDK to create and use them. There are code examples for several types of basic transactions, including:

* Asset issuance
* Simple payment
* Multi-asset payment
* Asset retirement

If you haven't already, you should first check out the [5-Minute Guide](../get-started/five-minute-guide.md). For advanced transaction features, see [Multiparty Trades](../build-applications/multiparty-trades.md).

### Sample Code

All code samples in this guide can be viewed in a single, runnable script. Available languages:

- [Java](../examples/java/TransactionBasics.java)
- [Ruby](../examples/ruby/transaction_basics.rb)

## Creating transactions

Creating a transaction consists of three steps:

1. **Build transaction**: Define what the transaction is supposed to do: issue new units of an asset, spend assets held in an account, control assets with an account, etc.
2. **Sign transaction**: Authorize the spending of assets or the issuance of new asset units using private keys.
3. **Submit transaction**: Submit a complete, signed transaction to the blockchain, and propagate it to other cores on the network.

### Build transaction

Rather than forcing you to manipulate inputs, outputs and change directly, the Chain Core API allows you to build transactions using a list of high-level **actions**.

There are seven types of actions:

Action                                  | Description
----------------------------------------|------------------------------------------------------------------------------------
Issue                                   | Issues new units of a specified asset.
Spend from account                      | Spends units of a specified asset from a specified account. Automatically handles locating outputs with enough units, and the creation of change outputs.
Spend an unspent output from an account | Spends an entire, specific unspent output in an account. Change must be handled manually, using other actions.
Control with account                    | Receives units of a specified asset into a specified account.
Control with program                    | Receives units of an asset into a specificed control program. Used when making a payment to an external party/account in another Chain Core.
Retire                                  | Retires units of a specified asset.
Set transaction reference data          | Sets arbitrary reference data on the transaction.

#### Reference data

You can annotate transactions with arbitrary reference data, which will be committed immutably to the blockchain alongside other details of the transaction. Reference data can be specified for the entire transaction, as well as for each action.

Action-level metadata will surface in the relevant inputs and ouputs. For example, the sender and recipient in a simple payment may each wish to set reference data for the actions that are directly relevant to them.

### Sign transaction

In order for a transaction to be accepted into the blockchain, its inputs must contain valid signatures. For issuance inputs, the signature must correspond to public keys named in the issuance program. For spending inputs, the signature must correspond to the public keys named in the control programs of the outputs being spent.

Transaction signing provides the blockchain with its security. Strong cryptography prevents everyone--even the operators of the blockchain network--from producing valid transaction signatures without the relevant private keys.

The Chain Core SDK assumes that private keys are held within an HSM controlled by the user. The SDK includes an `HsmSigner` interface that communicates with HSMs to sign transactions. For development, each Chain Core provides a Mock HSM that can generate public/private keypairs and sign transactions. It is important to note that the Mock HSM does not provide the security of a real HSM and, in a production setting, the Chain Core does not hold private keys and never signs transactions.

### Submit transaction

Once a transaction is balanced and all inputs are signed, it is considered valid and can be submitted to the blockchain. The local core will forward the transaction to the generator, which adds it to the blockchain and propagates it to other cores on the network.

The Chain Core API does not return a response until either the transaction has been added to the blockchain and indexed by the local core, or there was an error. This allows you to write your programs in a linear fashion. In general, if a submission responds with success, the rest of your program may proceed with the guarantee that the transaction has been committed to the blockchain.

## Examples

### Asset issuance

Issue 1000 units of gold to Alice.

#### Within a Chain Core

$code issue-within-core ../examples/java/TransactionBasics.java ../examples/ruby/transaction_basics.rb

#### Between two Chain Cores

First, Bob creates a control program in his account, which he can send to the issuer of gold.

$code create-bob-issue-program ../examples/java/TransactionBasics.java ../examples/ruby/transaction_basics.rb

The issuer then builds, signs, and submits a transaction, sending gold to Bob's control program.

$code issue-to-bob-program ../examples/java/TransactionBasics.java ../examples/ruby/transaction_basics.rb

### Simple payment

Alice pays 10 units of gold to Bob.

#### Within a Chain Core

$code pay-within-core ../examples/java/TransactionBasics.java ../examples/ruby/transaction_basics.rb

#### Between two Chain Cores

First, Bob creates a control program in his account, which he can send to Alice.

$code create-bob-payment-program ../examples/java/TransactionBasics.java ../examples/ruby/transaction_basics.rb

Alice then builds, signs, and submits a transaction, sending gold to Bob's control program.

$code pay-between-cores ../examples/java/TransactionBasics.java ../examples/ruby/transaction_basics.rb

### Multi-asset payment

Alice pays 10 units of gold and 20 units of silver to Bob.

#### Within a Chain Core

$code multiasset-within-core ../examples/java/TransactionBasics.java ../examples/ruby/transaction_basics.rb

#### Between two Chain Cores

First Bob creates a control program in his account, which he can send to Alice.

$code create-bob-multiasset-program ../examples/java/TransactionBasics.java ../examples/ruby/transaction_basics.rb

Alice then builds, signs, and submits a transaction, sending gold and silver to Bob's control program.

$code multiasset-between-cores ../examples/java/TransactionBasics.java ../examples/ruby/transaction_basics.rb

### Asset retirement

Alice retires 50 units of gold from her account.

$code retire ../examples/java/TransactionBasics.java ../examples/ruby/transaction_basics.rb

### Multiparty trades

For examples of advanced transactions, such as trading multiple assets across multiple cores, see [Multiparty Trades](../build-applications/multiparty-trades.md).
