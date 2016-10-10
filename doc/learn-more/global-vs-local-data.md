# Global vs Local Data

## Introduction
There are two types of data in Chain Core - global data that is committed to the blockchain and local data that is private to the Chain Core. Transactions and assets are the only global objects. All other objects are local.

Additionally, Chain Core privately annotates global transaction and asset objects with relevant local data.

For field-specific details, see [API Objects](../reference/api-objects.md#).

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
Accounts tags are local data that provide convenient storage and enable [complex queries](./query-filters.md). For example:

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
Asset tags are a local asset definition used to [query the blockchain](./query-filters.md). This is useful if you do not wish to publish the asset definition on the blockchain, but rather distribute it out of band to relevant parties. It also enables you to add additional local data about an asset that you didn't create. For example:

```
{
    "internal_rating": "B"   
}
```

## Examples

### Create accounts with tags
```
new Account.Builder()
    .setAlias("alice")
    .addXpub(mainKey.xpub)
    .setQuorum(1)
    .addTag("type", "checking")
    .addTag("first_name", "Alice")
    .addTag("last_name", "Jones")
    .addTag("user_id", "12345")
    .addTag("status", "enabled")
    .create(context);

new Account.Builder()
    .setAlias("bob")
    .addXpub(mainKey.xpub)
    .setQuorum(1)
    .addTag("type", "checking")
    .addTag("first_name", "Bob")
    .addTag("last_name", "Smith")
    .addTag("user_id", "67890")
    .addTag("status", "enabled")
    .create(context);
```
### Create assets with tags and definition
```
new Asset.Builder()
    .setAlias("acme-bond")
    .addXpub(mainKey.xpub)
    .setQuorum(1)
    .addTag("internal_rating", "B")
    .addDefinitionField("type", "security");
    .addDefinitionField("sub-type", "corporate-bond");
    .addDefinitionField("entity", "Acme Inc.");
    .addDefinitionField("maturity", "2016-09-01T18:24:47+00:00");
    .create(context);
```

### Create transaction with transaction-level reference data

```
Transaction.Template issuanceTransaction = new Transaction.Builder()
    .addAction(new Transaction.Action.Issue()
        .setAssetAlias("acme-bond")
        .setAmount(100)
    ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("acme-bond")
        .setAmount(100)
    ).addAction(new Transaction.Action.SetTransactionReferenceData()
        .addReferenceDataField("external_reference", "12345");
    ).build(context);
```

### Create transaction with action-level reference data
```
Transaction.Template issuanceTransaction = new Transaction.Builder()
    .addAction(new Transaction.Action.Issue()
        .setAssetAlias("acme-bond")
        .setAmount(100)
    ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("acme-bond")
        .setAmount(100)
        .addReferenceDataField("external_reference", "12345");
    ).build(context);
```
