# Global vs Local Data

## Introduction

There are two types of data in Chain Core - global data that is committed to the blockchain and local data that is private to the Chain Core. Transactions and assets are the only global objects. All other objects are local.

Additionally, Chain Core privately annotates global transaction and asset objects with relevant local data.

For field-specific details, see [API Objects](/doc/reference/api-objects).

## User-supplied global data

The Transaction.Builder and Asset.Builder classes each expose methods for providing user-supplied data that is committed to the blockchain:

* Asset definition
* Transaction reference data

### Asset definition

An asset definition is data about an asset that is written to the blockchain when you first issue units of an asset. It can include any details about an asset that you want participants in the blockchain to see. For example:

```
{
    "type": "security",
    "sub-type": "corporate-bond",
    "entity": "Acme Inc.",
    "maturity": "2016-09-01T18:24:47+00:00"
}
```

### Transaction reference data

Transaction reference data is included in a transaction when it is written to the blockchain. It is useful for any type of external data related to the transaction. For example:

```
{
    "external_reference": "123456"
}
```
Transaction reference data can be included at the top level of a transaction, or on each action (input/output) when building a transaction.

## User-supplied local data

The `Account.Builder` and `Asset.Builder` classes each expose methods for providing user-supplied data that is saved privately in the Chain Core.

### Account and asset aliases

Aliases are user-supplied, unique identifiers. They enable convenient operations on the client API.

### Account tags

Accounts tags are local data that provide convenient storage and enable [complex queries](/doc/building-applications/query-filters). For example:

```
{
    "type": "checking",
    "first_name": "Alice",
    "last_name": "Jones",
    "user_id": "123456",
    "status": "enabled"
}
```

### Asset tags

Asset tags are a local asset definition used to [query the blockchain](/doc/building-applications/query-filters). This is useful if you do not wish to publish the asset definition on the blockchain, but rather distribute it out of band to relevant parties. It also enables you to add additional local data about an asset that you didn't create. For example:

```
{
    "internal_rating": "B"
}
```

## Examples

### Create accounts with tags

$code /doc/examples/java/GlobalVsLocalData.java create-accounts-with-tags

### Create assets with tags and definition

$code /doc/examples/java/GlobalVsLocalData.java create-asset-with-tags-and-definition

### Create transaction with transaction-level reference data

$code /doc/examples/java/GlobalVsLocalData.java build-tx-with-tx-ref-data

### Create transaction with action-level reference data

$code /doc/examples/java/GlobalVsLocalData.java build-tx-with-action-ref-data

[Download Code](/doc/examples/java/GlobalVsLocalData.java)
