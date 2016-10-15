# Transactions Guide

## Introduction

A blockchain consists of an immutable set of cryptographically linked transactions. Each transaction consists of one or more inputs and outputs. An input defines a source of asset units — either a new issuance, or existing asset units controlled by a control program in an output of a previous transaction. An output defines an amount of asset units from the inputs to be controlled by a new control program or retired. When an output is not retired and has not yet been used as an input to a new transaction, we refer to it as an unspent output.

A transaction can have many inputs and outputs that consist of many different types of assets, all transferring together as a single atomic operation.

### Asset balancing

The total amount of assets in the inputs of a transaction must equal the total amount of assets in the outputs. To create new asset units, we issue in an input and control them with one or more outputs. To transfer asset units, we spend them in an input and control them with one or more outputs. To retire assets units, we spend them in an input and retire them in an output. Any combination of inputs and outputs can be used in a single transaction as long as what goes in comes out.

### Consuming entire unspent outputs

When creating an input that spends an unspent output, the entire amount must be consumed. It is not possible to only consume part of the output. Therefore, if you do not wish to transfer the entire amount of the output to your intended recipient, you must make change back to your account. This is accomplished by spending the input to two separate outputs — one for the recipient with the amount you want to spend, and one for yourself, with the remaining amount of the output being spent. The Transaction.QueryBuilder automatically creates change outputs when using with the `SpendFromAccount` action.

### Combining unspent outputs

If you wish to spend a greater amount of asset units than exist in a single unspent output, you can spend multiple unspent outputs to a single output. When you use the `SpendFromAccount` action on the `Transaction.QueryBuilder`, Chain Core automatically selects the correct number of outputs to satisfy the amount you want to spend, and, as noted above, makes change back to your account if there is any remainder.

## Overview

This guide will walk you through the basic functions of a transaction:

* Create transaction
* List transactions
* Transaction consumers

as well as a few examples of different transactions:

* Asset issuance
* Simple payment
* Multi-asset payment
* Asset trade
* Asset retirement

This guide assumes you know the basic functions presented in the [5-Minute Guide](../getting-started/five-minute-guide).

## Create Transaction

Creating a transaction consists of three basic steps:

1. Build transaction
2. Sign transaction
3. Submit transaction

Depending on the number of parties involved in a transaction, steps 1 and 2 may occur several times before submitting to the blockchain.

### Build transaction

The `Transaction.Builder` method is used to build new transactions. There are 7 actions that can be provided:

| Action                      | Description                                                                                                                                  |
|-----------------------------|----------------------------------------------------------------------------------------------------------------------------------------------|
| Issue                       | Issues new units of a specified asset.                                                                                                       |
| SpendFromAccount            | Spends units of a specified asset from a specified account. Automatically handles the creation of change outputs.                            |
| SpendAccountUnspentOutput   | Spends an entire unspent output in an account. Change must be handled manually by creating an additional ControlWithAccount action.          |
| ControlWithAccount          | Receives units of a specified asset into a specified account.                                                                                |
| ControlWithProgram          | Receives units of a specified asset into a control program. Used when making a payment to an external party / account in another Chain Core. |
| Retire                      | Retires units of a specified asset.                                                                                                          |
| setTransactionReferenceData | Sets arbitrary reference data on the transaction.                                                                                            |

#### Reference data

Reference data is useful for annotating transactions with external data. In addition to `setTransactionReferenceData`, reference data can be set on each action using the `setReferenceData` method on the `Transaction.Action`. For example, the sender and recipient in a simple payment may each wish to set independent action level reference data on the transactions. This reference data is added to the `reference_data` field on each input/output created by the transaction.

Note: the `SpendFromAccount` action will duplicate the reference data on each input it uses to satisfy the amount.

## Sign transaction

The SDK includes an HSMSigner that communicates with HSMs to sign transactions. For development, the HSMSigner can communicate with the MockHSM built into Chain Core.

#### Multi-party transactions

By default, the HSMSigner will sign the transaction in such a way that it cannot be altered. However, some types of transactions require more than one party to build and sign a single transaction. For example, if Alice and Bob want to trade silver for gold, Alice might build and sign her half, and then pass it to Bob for completion.

To enable this functionality, you must call the `allowAdditionalActions` method when signing the transaction. The transaction will then be signed in such a way that all currently built actions must occur for the transaction to be valid, but your counterparty can add additional actions to complete the transaction. You can then sign the transaction with the guarantee that your actions cannot be changed by your counterparty.

## Submit transaction

Once a transaction is balanced and all inputs are signed, you can submit it to the blockchain using the `Transaction.submit()` method. Chain Core waits until the transaction in included in a block to respond with a success.

## List transactions

The `Transaction.QueryBuilder` retrieves transactions from the blockchain. By default, it returns a paginated list of all transactions, ordered from latest to earliest timestamp. Custom queries can be achieved using the following methods:

| Method             | Description                                                        |
|--------------------|--------------------------------------------------------------------|
| setStartTime       | Sets the latest transaction timestamp to include in results.       |
| setEndTime         | Sets the earliest transaction timestamp to include in results.     |
| setFilter          | Sets a filter on the results.                                      |
| addFilterParameter | Defines a value for the first undefined placeholder in the filter. |
| setAscending       | Orders the results from earliest to latest timestamp.              |

### Filters

The `setFilter` method allows filtering `Transaction.QueryBuilder` results by any field in the [transaction object](../reference/api-objects#transaction). For more information, see [Query Filters](../building-applications/query-filters).

### Examples

List all transactions involving Alice's account:

$code ../examples/java/Transactions.java list-alice-transactions

List all transactions involving the Core:

$code ../examples/java/Transactions.java list-local-transactions

## Transaction consumers

Chain Core offers a mechanism by which you can automatically process new relevant transactions as they are committed to the blockchain. For example, if you create a control program in an account and give it to an external party, you may wish to take some action in your application when the payment arrives.

The steps are as follows:

1. Create a transaction feed (with an optional filter), marking a place in the time ordered list of transactions.
2. Ask the transaction feed for the next transaction (which will return the oldest unconsumed transaction).
3. Process the transaction in your application logic.
4. Acknowledge the transaction.
5. Return to Step 2.

If there is no next transaction, the feed will hold the connection open (via HTTP long polling) until the next transaction arrives.

### Example

We will process new payments into Alice's account as they arrive.

First, create a new transaction feed:

$code ../examples/java/Transactions.java create-feed

Then, process and acknowledge each new transaction as it arrives:

$code ../examples/java/Transactions.java process-feed

## Basic transaction examples

### Asset issuance

Issue 1000 units of gold to Alice.

#### Within a Chain Core

$code ../examples/java/Transactions.java issue-within-core

#### Between two Chain Cores

First Bob creates a control program in his account, which he can send to the issuer of gold.

$code ../examples/java/Transactions.java create-bob-issue-program

The issuer then builds, signs, and submits a transaction, sending gold to Bob's control program.

$code ../examples/java/Transactions.java issue-to-bob-program

### Simple payment

Alice pays 10 units of gold to Bob.

#### Within a Chain Core

$code ../examples/java/Transactions.java pay-within-core

#### Between two Chain Cores

First Bob creates a control program in his account, which he can send to Alice.

$code ../examples/java/Transactions.java create-bob-payment-program

Alice then builds, signs, and submits a transaction, sending gold to Bob's control program.

$code ../examples/java/Transactions.java pay-between-cores

### Multi-asset payment

Alice pays 10 units of gold and 20 units of silver to Bob.

#### Within a Chain Core

$code ../examples/java/Transactions.java multiasset-within-core

#### Between two Chain Cores
First Bob creates a control program in his account, which he can send to Alice.

$code ../examples/java/Transactions.java create-bob-multiasset-program

Alice then builds, signs, and submits a transaction, sending gold and silver to Bob's control program.

$code ../examples/java/Transactions.java multiasset-between-cores

### Asset trade

Alice trades 10 units of gold with Bob in return for 20 units of silver.

#### Within a Chain Core

$code ../examples/java/Transactions.java trade-within-core

#### Between two Chain Cores

We first build a transaction whereby Alice proposes to trade 10 units of gold for 20 units of silver.

$code ../examples/java/Transactions.java build-trade-a

The transaction builder constructs the transaction such that spending 10 units of gold from Alice's account *requires* 20 units of silver to simultaneously be received into Alice's account. We can then sign this transaction with the key used to create Alice's account to authorize Alice's portion of the proposed trade.

$code ../examples/java/Transactions.java sign-trade-a

The partial transaction can now be sent to Bob. Bob builds onto the transaction to satisfy the trade offer.

$code ../examples/java/Transactions.java build-trade-b

The complete transaction can now be signed with the key used to create Bob's account.

$code ../examples/java/Transactions.java sign-trade-a

Finally, Bob can submit the transaction to the blockchain to execute the trade.

$code ../examples/java/Transactions.java submit-trade

### Asset retirement

Alice retires 50 units of gold from her account.

$code ../examples/java/Transactions.java retire

[Download Code](../examples/java/Transactions.java)
