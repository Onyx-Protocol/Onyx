# Queries

## Introduction

Data structures in the Chain Core API are represented as key-value JSON objects. This includes local objects, such as accounts and keys, and global objects, such as transactions and assets. To retrieve data, you perform a query with optional parameters. By default, each query returns a time-ordered list of objects beginning with the most recent.

### Filters

Filters allow narrowing results to those matching a set of supplied parameters.

A filter is composed of one or more **terms**, with multiple terms joined with `AND` and `OR`. Each term contains a **property**, **operator**, and **value**. Each term targets a specific field in the key-value (JSON) object (see [API Objects](../reference/api-objects.md)). Terms can be grouped together in a **scope** to target a specific array of sub-objects within an object.

For example, to list transactions where a specific account spends a specific asset, you would create a filter with two terms, scoped to the inputs:

```
inputs(account_alias='alice' AND asset_alias='gold')
```

#### Properties

Any field in a JSON object can be used as a filter property. To use a field that is nested within another field, provide the path to it, starting with the outermost parent object. For example:

```
asset_definition.issuer.name
```

#### Operators

Filters currently support only the `=` operator.

#### Scope

The transaction object contains an array of other objects: an `inputs` array and an `outputs` array. The `inputs()` and `outputs()` filter scopes allow targeting a specific object within those arrays.

For example, the following will return transactions where Alice sent gold to Bob:

```
inputs(account_alias='alice' AND asset_alias='gold') AND outputs(account_alias='bob' AND asset_alias='gold')
```

### Additional parameters

Transaction queries accept time parameters to limit the results within a time window.

| Method             | Description                                                    |
|--------------------|----------------------------------------------------------------|
| setStartTime       | Sets the earliest transaction timestamp to include in results. |
| setEndTime         | Sets the latest transaction timestamp to include in results.   |

Balance and unspent output queries accept a timestamp parameter to report ownership at a specific moment in time.

| Method             | Description                                                                |
|--------------------|----------------------------------------------------------------------------|
| setTimestamp       | Sets a timestamp at which to calculate balances or return unspent outputs. |

### Special Case: Balance queries

Any balance on the blockchain is simply a summation of unspent outputs. For example, the balance of Alice’s account is a summation of all the unspent outputs whose control program was created from the keys in Alice’s account.

Unlike other queries in Chain Core, balance queries do not return Chain Core objects, only simple sums over the `amount` fields in a specified list of unspent output objects.

##### Sum By

Balance sums are totalled by `asset_id` and `account_id` by default, but it is also possible to query more complex sums. For example, if you have a network of counterparty-issued IOUs, you may wish to calculate the account balance of all IOUs from different counterparties that represent the same underlying currency.

## Overview

This guide will walk you through several examples of queries:

* Transactions
* Assets
* Accounts
* Unspent Outputs
* Balances

### Sample Code

All code samples in this guide are extracted from a single Java file.

<a href="../examples/java/Queries.java" class="downloadBtn btn success" target="\_blank">View Sample Code</a>

## Transactions

List all transactions involving Alice’s account:

$code ../examples/java/Queries.java list-alice-transactions

List all transactions involving the local Core:

$code ../examples/java/Queries.java list-local-transactions

## Assets

List all assets created in the local Core:

$code ../examples/java/Queries.java list-local-assets

List all assets with `USD` as the `currency` in the asset definition:

$code ../examples/java/Queries.java list-usd-assets

## Accounts

List all accounts with `checking` as the `type` in the account tags:

$code ../examples/java/Queries.java list-checking-accounts

## Unspent Outputs

List all unspent outputs controlled by Alice’s account:

$code ../examples/java/Queries.java list-alice-unspents

## Balances

List the asset IOU balances in Bank1’s account:

$code ../examples/java/Queries.java account-balance

Get the circulation of the Bank 1 USD IOU on the blockchain:

$code ../examples/java/Queries.java usd-iou-circulation

List the asset IOU balances in Bank1’s account, summed by currency:

$code ../examples/java/Queries.java account-balance-sum-by-currency
