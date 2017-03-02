# Chain Ruby SDK

## 1.0.2 (March 2, 2017)

* Relax minimum Ruby version requirement from 2.1 to 2.0. While the Ruby SDK is now compatible with Ruby 2.0, we strongly recommend using Ruby 2.1 or greater, since Ruby 2.0 has reached end-of-life and is no longer receiving critical security updates.

## 1.1.0 (February 24, 2017)

This release is a minor version update, and contains **new features** and **deprecations**. It is not compatible with cored 1.0.x; please upgrade cored before updating your SDKs.

### New object: receivers

Chain Core 1.1.x introduces the concept of a **receiver**, a cross-core payment primitive that supersedes the Chain Core 1.0.x pattern of creating and paying to control programs. Control programs still exist in the Chain protocol, but are no longer used directly to facilitate cross-core payments.

A receiver wraps a control program with other pieces of payment-related metadata, such as expiration dates. Receivers provide the basis for future payment features, such as the transfer of blinding factors for encrypted outputs, as well as off-chain proof of payment via X.509 certificates or some other cryptographic authentication scheme.

Initially, receivers consist of a **control program** and an **expiration date**. Transactions that pay to a receiver after the expiration date may not be tracked by Chain Core, and application logic should regard such payments as invalid. As long as both the payer and payee do not tamper with receiver objects, the Chain Core API will ensure that transactions that pay to expired receivers will fail to validate.

Working with receivers is very similar to working with control programs, and should require only small adjustments to your application code.

#### Creating receivers

The `create_control_program` method is **deprecated**. Instead, use `create_receiver`.

##### Deprecated (1.0.x)

```
cp = client.accounts.create_control_program(
  alias: 'alice'
)
```

##### New (1.1.x)

You can create receivers with an expiration time. This parameter is optional and defaults to 30 days into the future.

```
receiver = client.accounts.create_receiver(
  account_alias: 'alice',
  expires_at: '2017-01-01T00:00:00Z'
)
```

#### Using receivers in transactions

The `control_with_program` transaction builder method is **deprecated**. Use `control_with_receiver` instead.

##### Deprecated (1.0.x)

```
template = client.transactions.build do |builder|
  builder.control_with_program(
    control_program: control_program.control_program,
    asset_alias: 'gold',
    amount: 1
  )
  ...
end
```

##### New (1.1.x)

```
template = client.transactions.build do |builder|
  builder.control_with_receiver(
    receiver: receiver,
    asset_alias: 'gold',
    amount: 1
  )
  ...
end
```

Transactions that pay to expired receivers will fail during validation, i.e., while they are being submitted.

### New output property: unique IDs

In Chain Core 1.0.x, transaction outputs were addressed using a compound value consisting of a transaction ID and output position. Chain Core 1.1.x introduces an ID property for each output that is unique across the blockchain.

#### Updates to data structures

##### Transaction outputs and unspent outputs

Transaction output objects and unspent outputs now have an `id` property, which is unique for that output across the history of the blockchain.

```
puts tx.outputs.first.id
puts utxo.id
```

##### Transaction inputs

The `spent_output` property on `Chain::Transaction::Input` is **deprecated**. Use `spent_output_id` instead.

```
puts tx.inputs.first.spent_output_id
```

#### Spending unspent outputs in transactions

The `spend_account_unspent_output` transaction builder method now accepts an `output_id` parameter. The `transaction_id` and `position` parameters are **deprecated**.

##### Deprecated (1.0.x)

```
template = client.transactions.build do |builder|
  builder.spend_account_unspent_output(
    transaction_id: 'abc123',
    position: 0
  )
  ...
end
```

##### New (1.1.x)

```
template = client.transactions.build do |builder|
  builder.spend_account_unspent_output(
    output_id: 'xyz789'
  )
  ...
end
```

#### Querying previous transactions

To retrieve transactions that were partially consumed by a given transaction input, you can query against a specific output ID.

##### Deprecated (1.0.x)

```
client.transactions.query(
  filter: 'id=$1',
  filter_parameters: [spending_tx.inputs.first.spent_output.transaction_id]
) do |tx|
  ...
end
```

##### New (1.1.x)

```
client.transactions.query(
  filter: 'outputs(id=$1)',
  filter_parameters: [spending_tx.inputs.first.spent_output_id]
) do |tx|
  ...
end
```

## 1.0.2 (February 21, 2017)

* Syntax compatibility update

## 1.0.1 (January 24, 2017)

* Set minimum Ruby version requirement to 2.1
* Enhanced transaction feed API support
* Fixed issue reading attributers with array getter syntax (@donce in [#422](https://github.com/chain/chain/pull/422))

## 1.0.0 (November 17, 2016)

* Initial release
