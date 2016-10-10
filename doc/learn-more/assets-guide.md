# Assets Guide

## Introduction
An asset is a set of fungible units that can be issued on a blockchain to represent any type of value. Once on a blockchain, units of an asset can be transacted directly between parties without the involvement of the issuer. The issuer can issue as many units as they want, as many times as they want.

Each asset has a globally unique asset ID that is derived from an issuance program. The issuance program defines a set of private keys and a quorum of signatures that must be provided to issue units of the assets. Chain Core automatically creates the issuance program when the asset is created.

Each asset can optionally include an asset definition, consisting of arbitrary key-value data. The asset definition is committed to the blockchain for all participants to see. Additionally, an asset can be tagged locally with private data for convenient queries and operations. For more information, see [Global vs. local data](./global-vs-local-data.md).

## Overview
This guide will walk you through the basic functions of an asset:

* Create asset
* List assets (by asset definition, tags, and origin)
* Issue asset units to a local account (in the same Chain Core)
* Issue asset units to an external party
* Trade asset units by issuing to an external party
* Retire asset units
* List transactions (for issuance, transfer, and retirement)
* Get asset circulation

This guide assumes you know the basic functions presented in the [5-Minute Guide](./five-minute-guide).

## Create asset
Creating an asset defines the asset object locally in the Chain Core. It does not exist on the blockchain until units are issued in a transaction.

* An `alias` is an optional, user-supplied, unique identifier that you can use to operate on the asset. We will use this later to build a transaction issuing units of the asset.
* The `quorum` is the threshold of keys that must sign a transaction to spend asset units controlled by the account.
* A `definition` is global data about the asset that is visible in the blockchain. We will create several fields in the definition.
* A `tag` is an optional key-value field used for arbitrary storage or queries. This data is local to the Chain Core and *not* visible in the blockchain. We will add several tags.

Create an asset for Acme Common stock
```java
// Create the asset definition
Map<String, Object> acmeCommonDef = new HashMap<>();
acmeCommonDef.put("issuer", "Acme Inc.");
acmeCommonDef.put("type", "security");
acmeCommonDef.put("subtype", "private");
acmeCommonDef.put("class", "common");

// Build the asset
new Asset.Builder()
  .setAlias("acme_common")
  .addRootXpub(key.xpub)
  .setQuorum(1)
  .addTag("internal_rating", "1")
  .setDefinition(acmeCommonDef)
  .create(context);
```

Create an asset for Acme Preferred stock
```java
// Create the asset definition
Map<String, Object> acmePreferredDef = new HashMap<>();
acmePreferredDef.put("issuer", "Acme Inc.");
acmePreferredDef.put("type", "security");
acmePreferredDef.put("subtype", "private");
acmePreferredDef.put("class", "perferred");

// Build the asset
new Asset.Builder()
  .setAlias("acme_preferred")
  .addRootXpub(key.xpub)
  .setQuorum(1)
  .addTag("internal_rating", "2")
  .setDefinition(acmePreferredDef)
  .create(context);
```

## List assets
Chain Core keeps a list of all assets in the blockchain, whether or not they were issued in the Chain Core. Each asset can be locally annotated with an alias and tags to enable efficient actions and intelligent queries. Note: local data is not present in the blockchain. For more information, see: [Local vs. Global Data](#).

To list all assets created in the Core, we build an assets query, filtering to the `origin` tag.
```java
Asset.Items localAssets = new Asset.QueryBuilder()
  .setFilter("origin=$1")
  .addFilterParameter("local")
  .execute(context);
```

To list all assets defined as preferred stock of a private security, we build an assets query, filtering to several tags.
```java
Assets.Items common = new Asset.QueryBuilder()
  .setFilter("definition.type=$1 AND definition.subtype=$2 AND definition.class=$3")
  .addFilterParameter("security")
  .addFilterParameter("private")
  .addFilterParameter("preferred")
  .execute(context);
```

## Issue asset units to a local account
To issue units of an asset into an account within the Chain Core, we can build a transaction using an `asset_alias` and an `account_alias`.

We first build a transaction issuing 1000 units of Acme Common stock to the Acme treasury account.

```java
Transaction.Template issuanceTransaction = new Transaction.Builder()
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias("acme_common")
    .setAmount(1000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias("acme_treasury")
    .setAssetAlias("acme_common")
    .setAmount(1000)
  ).build(context);
```

Once we have built the transaction, we need to sign it with the key used to create the Acme Common stock asset.

```java
Transaction.Template signedIssuanceTransaction = HsmSigner.sign(issuanceTransaction);
```

Once we have signed the transaction, we can submit it for inclusion in the blockchain.
```java
Transaction.submit(context, signedIssuanceTransaction);
```

## Issue asset units to an external party
If you wish to issue asset units to an external party, you must first request a control program from them. You can then build, sign, and submit a transaction issuing asset units to their control program.

We will issue 2000 units of Acme Common stock to an external party.

```java
Transaction.Template issuanceTransaction2 = new Transaction.Builder()
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias("acme_common")
    .setAmount(2000)
  ).addAction(new Transaction.Action.ControlWithProgram()
    .setControlProgram(externalProgram)
    .setAssetAlias("acme_common")
    .setAmount(2000)
  ).build(context);
Transaction.submit(context, HsmSigner.sign(issuanceTransaction2));
```

## Trade asset units by issuing to an external party
Chain Core enables risk-free bilateral trades. The steps are as follows:

1. The first party builds a partial transaction proposing the trade
2. The first party signs the partial transaction
3. The first party sends the partial transaction to the second party
4. The second party builds onto the partial transaction to satisfy the proposed trade
5. The second party signs the complete transaction
6. The second party submits the transaction to the blockchain

We first build a transaction whereby Acme proposes to issue 1000 units of Acme Common stock to Bob for $50,000. Note: the USD asset is denominated in cents, so the amount is 5,000,000.
```java
Transaction.Template tradeProposal = new Transaction.Builder()
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias("acme_common")
    .setAmount(1000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias("acme_treasury")
    .setAssetAlias("USD")
    .setAmount(5000000)
  ).build(context);
```

The transaction builder constructs the transaction such that issuing 1000 units of Acme Common stock *requires* 5,000,000 units of USD (cents) to simultaneously be received into Acme's account. We can then sign this transaction with the key used to create the Acme Common stock asset to authorize Acme's portion of the proposed trade.

```
Transaction.Template signedTradeProposal = HsmSigner.sign(tradeProposal);
```

The partial transaction can now be sent to Bob. Bob builds onto the transaction to satisfy the trade offer. Note: Bob has locally aliased the Acme Common stock asset as `acme_common-stock`.
```java
Transaction.Template tradeTransaction = new Transaction.Builder()
  .setRawTransaction(signedTradeProposal.rawTransaction)
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias("bob")
    .setAssetAlias("USD")
    .setAmount(5000000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias("bob")
    .setAssetAlias("acme_commmon")
    .setAmount(1000)
  ).build(context);
```

The complete transaction can now be signed with the key used to create Bob's account.
```java
Transaction.Template signedTradeTransaction = HsmSigner.sign(tradeTransaction);
```

Finally, Bob can submit the transaction to the blockchain to execute the trade.

```java
Transaction.submit(context, signedTradeTransaction);
```

### Retire asset units
To retire units of an asset from an account, we can build a transaction using an `account_alias` and `asset_alias`.

We first build a transaction retiring 50 units of Acme Common stock from Acme's treasury account.

```java
Transaction.Template retirementTransaction = new Transaction.Builder()
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias("acme_treasury")
    .setAssetAlias("acme_common")
    .setAmount(50)
  ).addAction(new Transaction.Action.Retire()
    .setAssetAlias("acme_common")
    .setAmount(50)
  ).build(context);
```

Once we have built the transaction, we need to sign it with the key used to create Acme's treasury account.

```java
Transaction.Template signedRetirementTransaction = HsmSigner.sign(retirementTransaction);
```

Once we have signed the transaction, we can submit it for inclusion in the blockchain.
```java
Transaction.submit(context, signedRetirementTransaction);
```

## List asset transactions
Chain Core keeps a time-ordered list of all transactions in the blockchain. These transactions are locally annotated with asset aliases and asset tags to enable intelligent queries. Note: local data is not present in the blockchain. For more information, see: [Local vs. Global Data](#).

### Issuance transactions
To list transactions where Acme Common stock was issued, we build an assets query, filtering to inputs with the `issue` action and the Acme Common stock `asset_alias`.

```java
Transaction.Items transactions = new Transaction.QueryBuilder()
  .setFilter("inputs(action=$1 AND asset_alias=$2)")
  .addFilterParameter("issue")
  .addFilterParameter("acme_common")
  .execute(context);
```
### Transfer transactions
To list transactions where Acme Common stock was transferred, we build an assets query, filtering to inputs with the `spend` action and the Acme Common stock `asset_alias`.

```java
Transaction.Items transactions = new Transaction.QueryBuilder()
  .setFilter("inputs(action=$1 AND asset_alias=$2)")
  .addFilterParameter("spend")
  .addFilterParameter("acme_common")
  .execute(context);
```
### Retirement transactions
To list transactions where Acme Common stock was retired, we build an assets query, filtering to outputs with the `retire` action and the Acme Common stock `asset_alias`.

```java
Transaction.Items transactions = new Transaction.QueryBuilder()
  .setFilter("outputs(action=$1 AND asset_alias=$2)")
  .addFilterParameter("retire")
  .addFilterParameter("acme_common")
  .execute(context);
```

## Get asset circulation
The circulation of an asset is the sum of all asset units controlled by any control program (existing in unspent_outputs) in the blockchain.

To list the circulation of Acme Common stock, we build a balance query, filtering to the Acme Common stock `asset_alias`.

```java
Balance.Items balances = new Balance.QueryBuilder()
  .setFilter("asset_alias=$1")
  .addFilterParameter("acme_common")
  .execute(context);
```

To list the circulation of all classes of Acme stock, we build a balance query, filtering to the `issuer` field in the `definition`.

```java
Balance.Items balances = new Balance.QueryBuilder()
  .setFilter("asset_definition.entity=$1")
  .addFilterParameter("Acme Inc.")
  .execute(context);
```

To list all the control programs that hold a portion of the circulation of Acme Common stock, we build an unspent outputs query, filtering to the Acme Common stock `asset_alias`.

```java
UnspentOutput.Items acmeCommonUnspentOutputs = new UnspentOutput.QueryBuilder()
  .setFilter("asset_alias='$1'")
  .addFilterParameter("acme_common")
  .execute(context);
```
