# Unspent Outputs Guide

## Introduction

Each new transaction in the blockchain consumes some unspent outputs and creates others. An output is considered unspent when it has not yet been used as an input to a new transaction. All asset units on a blockchain exist in the unspent output set.

## Overview

This guide will walk you through the basic functions of an unspent output:

* [List unspent outputs](#list-unspent-outputs)
* [Spend unspent outputs](#spend-unspent-outputs)

### Sample Code

All code samples in this guide can be viewed in a single, runnable script. Available languages:

- [Java](../examples/java/UnspentOutputs.java)
- [Ruby](../examples/ruby/unspent_outputs.rb)

## List unspent outputs

List all unspent outputs in Alice's account:

$code alice-unspent-outputs ../examples/java/UnspentOutputs.java ../examples/ruby/unspent_outputs.rb

List all unspent outputs of the gold asset:

$code gold-unspent-outputs ../examples/java/UnspentOutputs.java ../examples/ruby/unspent_outputs.rb

## Spend unspent outputs

When building a transaction with the “spend from account” action type, Chain Core automatically selects one or more unspent outputs sufficient to cover the amount to be spent, and automatically returns any excess to your account by adding a change output to the transaction. However, if you want to spend specific unspent outputs, you can use the “spend unspent output from account” action type. You do not specify an amount or asset for the action, but rather spend the entire amount of the asset controlled in the unspent output. Unlike “spend from account,” this action type does not automatically make change. If you wish to spend only a portion of the unspent output, you must explicitly make change back to your account by adding a “control with account” action.

## Example

### Spend entire unspent output

Given the following unspent output in Alice's account:

```
{
  "transaction_id": "ad8e8aa37b0969ec60151674c821f819371152156782f107ed49724b8edd7b24",
  "position": 1,
  "asset_id": "d02e4a4c3b260ae47ba67278ef841bbad6903bda3bd307bee2843246dae07a2d",
  "asset_alias": "gold",
  "amount": 100,
  "account_id": "acc0KFJCM6KG0806",
  "account_alias": "alice",
  "control_program": "766baa2056d4bfb5fcc08a13551099e596ebb9982d2c913285ef6751767fda0d111ddc3f5151ad696c00c0",
}
```

Build a transaction spending all units of gold in the unspent output to Bob's account:

$code build-transaction-all ../examples/java/UnspentOutputs.java ../examples/ruby/unspent_outputs.rb

### Spend partial unspent output

Given the following unspent output in Alice's account:

```
{
  "transaction_id": "ad8e8aa37b0969ec60151674c821f819371152156782f107ed49724b8edd7b24",
  "position": 1,
  "asset_id": "d02e4a4c3b260ae47ba67278ef841bbad6903bda3bd307bee2843246dae07a2d",
  "asset_alias": "gold",
  "amount": 100,
  "account_id": "acc0KFJCM6KG0806",
  "account_alias": "alice",
  "control_program": "766baa2056d4bfb5fcc08a13551099e596ebb9982d2c913285ef6751767fda0d111ddc3f5151ad696c00c0",
}
```

Build a transaction spending 40 units of gold in the unspent output to Bob's account, and spending 60 units back to Alice's account as change:

$code build-transaction-partial ../examples/java/UnspentOutputs.java ../examples/ruby/unspent_outputs.rb
