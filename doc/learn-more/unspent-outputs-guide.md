# Unspent Outputs Guide

## Introduction

This guide will walk you through the basic functions of an unspent output:

1. List unspent outputs
2. Spend unspent outputs


## List unspent outputs
All asset units on the blockchain exist in the unspent output set. Any balance on the blockchain is simply a summation of unspent outputs. For example, the balance of Alice's account is a summation of all the unspent outputs whose control program was created with Alice's account.

### Examples

List all unspent outputs in Alice's account:

```
UnspentOutput.Items aliceUnspentOutputs = new UnspentOutput.QueryBuilder()
  .setFilter("account_alias=$1")
  .addFilterParameter("alice")
  .execute(context);
```

List all unspent outputs of the Gold asset:

```
UnspentOutput.Items goldUnspentOutputs = new UnspentOutput.QueryBuilder()
  .setFilter("asset_alias=$1")
  .addFilterParameter("gold")
  .execute(context);
```

## Spend unspent outputs

When building a transaction with the `SpendFromAccount` action in the `Transaction.Builder`, Chain Core automatically selects unspent outputs. However, if you want to manually spend unspent outputs, you can use the `SpendAccountUnspentOutput` action. Unlike the `SpendFromAccount` action, the  `SpendAccountUnspentOutput` action does not automatically make change. Therefore, you do not pass an amount or asset to the action, but rather spend the entire amount of the asset controlled in the unspent output. If you wish to only spend a portion of the unspent output, you must manually make change back to your account.

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


```
Transaction.Template spend1 = new Transaction.Builder()
  .addAction(new Transaction.Action.SpendAccountUnspentOutput()
    .setTransactionId("ad8e8aa37b0969ec60151674c821f819371152156782f107ed49724b8edd7b24")
    .setPosition(1)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias("bob")
    .setAssetAlias("gold")
    .setAmount(100)
  ).build(context);
```

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


```
Transaction.Template spend1 = new Transaction.Builder()
  .addAction(new Transaction.Action.SpendAccountUnspentOutput()
    .setTransactionId("ad8e8aa37b0969ec60151674c821f819371152156782f107ed49724b8edd7b24")
    .setPosition(1)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias("bob")
    .setAssetAlias("gold")
    .setAmount(40)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias("alice")
    .setAssetAlias("gold")
    .setPurpose("change")
    .setAmount(60)
  ).build(context);
```
