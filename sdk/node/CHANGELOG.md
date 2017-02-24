# Chain Node.js SDK

## 1.1.0 (February 24, 2017)

This release is a minor version update, and contains **new features** and **deprecations**. It is not compatible with cored 1.0.x; please upgrade cored before updating your SDKs.

### New object: receivers

Chain Core 1.1.x introduces the concept of a **receiver**, a cross-core payment primitive that supersedes the Chain Core 1.0.x pattern of creating and paying to control programs. Control programs still exist in the Chain protocol, but are no longer used directly to facilitate cross-core payments.

A receiver wraps a control program with other pieces of payment-related metadata, such as expiration dates. Receivers provide the basis for future payment features, such as the transfer of blinding factors for encrypted outputs, as well as off-chain proof of payment via X.509 certificates or some other cryptographic authentication scheme.

Initially, receivers consist of a **control program** and an **expiration date**. Transactions that pay to a receiver after the expiration date may not be tracked by Chain Core, and application logic should regard such payments as invalid. As long as both the payer and payee to do not tamper with receiver objects, the Chain Core API will ensure that transactions that pay to expired receivers will fail to validate.

Working with receivers is very similar to working with control programs, and should require only small adjustments to your application code.

#### Creating receivers

The `createControlProgram` method is **deprecated**. Instead, use `createReceiver`.

##### Deprecated (1.0.x)

```
controlProgramPromise = client.accounts.createControlProgram({
  alias: 'alice'
})
```

##### New (1.1.x)

You can create receivers with an expiration time. This parameter is optional and defaults to 30 days into the future.

```
receiverPromise = client.accounts.createReceiver({
  accountAlias: 'alice',
  expiresAt: '2017-01-01T00:00:00Z'
})
```

#### Using receivers in transactions

The `controlWithProgram` transaction builder method is **deprecated**. Use `controlWithReceiver` instead.

##### Deprecated (1.0.x)

```
templatePromise = client.transactions.build(builder => {
  builder.controlWithProgram({
    controlProgram: controlProgram.controlProgram,
    assetAlias: 'gold',
    amount: 1
  })
  ...
})
```

##### New (1.1.x)

```
templatePromise = client.transactions.build(builder => {
  builder.controlWithReceiver({
    reciever: receiver,
    assetAlias: 'gold',
    amount: 1
  })
  ...
})
```

Transactions that pay to expired receivers will fail during validation, i.e., while they are being submitted.

### New output property: unique IDs

In Chain Core 1.0.x, transaction outputs were addressed using a compound value consisting of a transaction ID and output position. Chain Core 1.1.x introduces an ID property for each output that is unique across the blockchain.

#### Updates to data structures

##### Transaction outputs and unspent outputs

Transaction output objects and unspent outputs now have an `id` property, which is unique for that output across the history of the blockchain.

```
console.log(tx.outputs[0].id)
console.log(utxo.id)
```

##### Transaction inputs

The `spentOutput` property on transaction intputs is **deprecated**. Use `spentOutputId` instead.

```
console.log(tx.inputs[0].spentOutputId)
```

#### Spending unspent outputs in transactions

The `spendUnspentOutput` method now takes an `outputId` parameter. The `transactionId` and `position` parmeters are **deprecated**.

##### Deprecated (1.0.x)

```
templatePromise = client.transactions.build(builder => {
  builder.spendUnspentOutput({
    transactionId: 'abc123',
    position: 0
  })
  ...
})
```

##### New (1.1.x)

```
templatePromise = client.transactions.build(builder => {
  builder.spendAccountUnspentOutput({
    outputId: 'xyz789'
  })
  ...
})
```

#### Querying previous transactions

To retrieve transactions that were partially consumed by a given transaction input, you can query against a specific output ID.

##### Deprecated (1.0.x)

```
client.transactions.queryAll({
  filter: 'id=$1',
  filterParameters: [spendingTx.inputs[0].spentOutput.transactionId]
}, (tx, next, done, fail) => {
  ...
})
```

##### New (1.1.x)

```
client.transactions.queryAll({
  filter: 'outputs(id=$1)',
  filterParameters: [spendingTx.inputs[0].spentOutputId]
}, (tx, next, done, fail) => {
  ...
})
```

## 1.0.2 (January 25, 2017)

* Use base URL and client token provided on initialization for MockHSM connection
* Allow users to instantiate `Connection` objects with `new chain.Connection()`

## 1.0.1 (January 24, 2017)

* README and documentation updates

## 1.0.0 (January 20, 2017)

* Initial release
