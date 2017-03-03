# Chain Java SDK changelog

## 1.1.1 (March 3, 2017)

You can now specify trusted server SSL certificates via a PEM-encoded file:

```
Client client = new Client.Builder()
  .setURL("https://example:443")
  .setTrustedCerts("path/to/certs.pem")
  .build();
```

## 1.1.0 (February 24, 2017)

This release is a minor version update, and contains **new features** and **deprecations**. It is not compatible with cored 1.0.x; please upgrade cored before updating your SDKs.

### New object: receivers

Chain Core 1.1.x introduces the concept of a **receiver**, a cross-core payment primitive that supersedes the Chain Core 1.0.x pattern of creating and paying to control programs. Control programs still exist in the Chain protocol, but are no longer used directly to facilitate cross-core payments.

A receiver wraps a control program with other pieces of payment-related metadata, such as expiration dates. Receivers provide the basis for future payment features, such as the transfer of blinding factors for encrypted outputs, as well as off-chain proof of payment via X.509 certificates or some other cryptographic authentication scheme.

Initially, receivers consist of a **control program** and an **expiration date**. Transactions that pay to a receiver after the expiration date may not be tracked by Chain Core, and application logic should regard such payments as invalid. As long as both the payer and payee do not tamper with receiver objects, the Chain Core API will ensure that transactions that pay to expired receivers will fail to validate.

Working with receivers is very similar to working with control programs, and should require only small adjustments to your application code.

#### Creating receivers

Creating control programs via the `ControlProgram.Builder` class is **deprecated**. Instead, use `Account.ReceiverBuilder`.

##### Deprecated (1.0.x)

```
ControlProgram controlProgram = new ControlProgram.Builder()
  .controlWithAccountByAlias("alice")
  .create(client);
```

##### New (1.1.x)

You can create receivers with an expiration time. This parameter is optional and defaults to 30 days into the future.

```
Receiver receiver = new Account.ReceiverBuilder()
  .setAccountAlias("alice")
  .setExpiresAt("2017-01-01T00:00:00Z")
  .create(client);
```

#### Using receivers in transactions

`Transaction.Action.ControlWithProgram` is **deprecated**. Use `Transaction.Action.ControlWithReceiver` instead.

##### Deprecated (1.0.x)

```
Transaction.Template template = new Transaction.Builder()
  .addAction(
    new Transaction.Action.ControlWithProgram()
      .setControlProgram(controlProgram.controlProgram)
      .setAssetAlias("gold")
      .setAmount(1)
  ).addAction(
    ...
  ).build(client);
```

##### New (1.1.x)

```
Transaction.Template template = new Transaction.Builder()
  .addAction(
    new Transaction.Action.ControlWithReceiver()
      .setReceiver(receiver)
      .setAssetAlias("gold")
      .setAmount(1)
  ).addAction(
    ...
  ).build(client);
```

Transactions that pay to expired receivers will fail during validation, i.e., while they are being submitted.

### New output property: unique IDs

In Chain Core 1.0.x, transaction outputs were addressed using a compound value consisting of a transaction ID and output position. Chain Core 1.1.x introduces an ID property for each output that is unique across the blockchain.

#### Updates to data structures

##### Transaction outputs and unspent outputs

Transaction output objects and unspent outputs now have an `id` property, which is unique for that output across the history of the blockchain.

```
Transaction tx;
UnspentOutput utxo;
System.out.println(tx.outputs.get(0).id);
System.out.println(utxo.id);
```

##### Transaction inputs

The `spentOutput` property on `Transaction.Input` is **deprecated**. Use `spentOutputId` instead.

```
Transaction tx;
System.out.println(tx.inputs.get(0).spentOutputId);
```

#### Spending unspent outputs in transactions

`Transaction.Action.SpendAccountUnspentOutput` now has a `setOutputId` method. The `setTransactionId` and `setPosition` methods are **deprecated**.

##### Deprecated (1.0.x)

```
Transaction.Template template = new Transaction.Builder()
  .addAction(
    new Transaction.Action.SpendAccountUnspentOutput()
      .setTransactionId("abc123")
      .setPosition(0)
  ).addAction(
    ...
  ).build(client);
end
```

##### New (1.1.x)

```
Transaction.Template template = new Transaction.Builder()
  .addAction(
    new Transaction.Action.SpendAccountUnspentOutput()
      .setOutputId("xyz789")
  ).addAction(
    ...
  ).build(client);
```

#### Querying previous transactions

To retrieve transactions that were partially consumed by a given transaction input, you can query against a specific output ID.

##### Deprecated (1.0.x)

```
Transaction.Items results = new Transaction.QueryBuilder()
  .setFilter("id=$1")
  .setFilterParameter(spendingTx.inputs.get(0).spentOutput.transactionId)
  .execute(client);
```

##### New (1.1.x)

```
Transaction.Items results = new Transaction.QueryBuilder()
  .setFilter("outputs(id=$1)")
  .setFilterParameter(spendingTx.inputs.get(0).spentOutputId)
  .execute(client);
```

## 1.0.1 (December 2, 2016)<a name="1.0.1"></a>

* Java 7 support

## 1.0.0 (October 24, 2016)

* Initial release
